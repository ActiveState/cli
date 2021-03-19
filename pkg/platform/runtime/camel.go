package runtime

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/unarchiver"
)

var _ EnvGetter = &CamelEnv{}
var _ Assembler = &CamelInstall{}

const envFile = "activestate.env.json"
const deleteMarker = "!#DELETE#!"

// CamelEnv holds all the meta-data necessary to activate a runtime
// environment for a Camel build
type CamelEnv struct {
	commitID   strfmt.UUID
	runtimeDir string
	env        map[string]string
}

// CamelInstall provides methods to download and install camel artifacts
type CamelInstall struct {
	CamelEnv
	artifacts []*HeadChefArtifact
}

// NewCamelEnv returns a new camel runtime assembler
// It filters the provided artifact list for use-able artifacts
func NewCamelEnv(commitID strfmt.UUID, cacheDir string) (*CamelEnv, error) {
	ce := &CamelEnv{commitID, cacheDir, map[string]string{}}
	return ce, nil
}

// NewCamelInstall creates a new camel installation
func NewCamelInstall(commitID strfmt.UUID, cacheDir string, artifacts []*HeadChefArtifact) (*CamelInstall, error) {
	ce, err := NewCamelEnv(commitID, cacheDir)
	if err != nil {
		return nil, err
	}
	ci := &CamelInstall{*ce, []*HeadChefArtifact{}}

	for _, artf := range artifacts {
		// filter artifacts
		if artf.URI == "" {
			continue
		}

		filename := filepath.Base(artf.URI.String())
		if !strings.HasSuffix(filename, ci.InstallerExtension()) || strings.Contains(filename, InstallerTestsSubstr) {
			continue
		}

		ci.artifacts = append(ci.artifacts, artf)
	}

	if len(ci.artifacts) == 0 {
		return ci, &ErrInvalidArtifact{locale.NewError("err_no_valid_artifact")}
	}

	return ci, nil
}

// InstallerExtension returns the expected file extension for archive file names
// We expect .zip for Windows and .tar.gz otherwise
func (ci *CamelInstall) InstallerExtension() string {
	if rt.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// Unarchiver initializes and returns an Unarchiver instance that is able to
// unpack the downloaded artifact archives.
func (ci *CamelInstall) Unarchiver() unarchiver.Unarchiver {
	if rt.GOOS == "windows" {
		return unarchiver.NewZip()
	}
	return unarchiver.NewTarGz()
}

// BuildEngine always returns Camel
func (ci *CamelInstall) BuildEngine() BuildEngine {
	return Camel
}

// DownloadDirectory returns the download directory for a given artifact
// Each artifact is downloaded into its own temporary directory
func (ci *CamelInstall) DownloadDirectory(artf *HeadChefArtifact) (string, error) {
	downloadDir, err := ioutil.TempDir("", "state-runtime-downloader")
	if err != nil {
		return downloadDir, errs.Wrap(err, "TempDir failed")
	}
	return downloadDir, nil
}

// ArtifactsToDownload returns the artifacts that we need to download for this project
// It filters out all artifacts for which the final installation directory does not include a completion marker yet
func (ci *CamelInstall) ArtifactsToDownload() []*HeadChefArtifact {
	return ci.artifacts
}

// PreInstall attempts to clean the runtime-directory.  Failures are only logged to rollbar and do not cause the installation to fail.
func (ci *CamelInstall) PreInstall() error {
	if fileutils.DirExists(ci.runtimeDir) {
		empty, err := fileutils.IsEmptyDir(ci.runtimeDir)
		if err != nil {
			logging.Error("Could not check if target runtime dir is empty, this could cause issues.. %v", err)
		} else if !empty {
			logging.Debug("Removing existing runtime")
			if err := os.RemoveAll(ci.runtimeDir); err != nil {
				logging.Error("Could not empty out target runtime dir prior to install, this could cause issues.. %v", err)
			}
		}
	}
	return nil
}

// PreUnpackArtifact ensures that the final installation directory exists and is
// useable.
// Note:  It will remove a previous installation
func (ci *CamelInstall) PreUnpackArtifact(artf *HeadChefArtifact) error {
	if fileutils.FileExists(ci.runtimeDir) {
		// install-dir exists, but is a regular file
		return &ErrInstallDirInvalid{locale.NewInputError("installer_err_installdir_isfile", "", ci.runtimeDir)}
	}

	if fileutils.DirExists(ci.runtimeDir) {
		// remove previous installation
		if err := os.RemoveAll(ci.runtimeDir); err != nil {
			return errs.Wrap(err, "failed to remove spurious previous installation")
		}
	}

	if err := fileutils.MkdirUnlessExists(ci.runtimeDir); err != nil {
		return err
	}

	return nil
}

// PostUnpackArtifact parses the metadata file, runs the Relocation function (if
// necessary) and moves the artifact to its final destination
func (ci *CamelInstall) PostUnpackArtifact(artf *HeadChefArtifact, tmpRuntimeDir string, archivePath string, cb func()) error {
	archiveName := strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))

	// the above only strips .gz, so account for .tar.gz use-case
	// it's fine to run this on windows cause those files won't end in .tar anyway
	archiveName = strings.TrimSuffix(archiveName, ".tar")

	// Detect the install dir (in the tarball)
	// Python runtimes on MacOS work where they are unarchived so we do not
	// need to do any detection of the install directory
	var tmpInstallDir string
	installDirs := strings.Split(constants.RuntimeInstallDirs, ",")
	for _, dir := range installDirs {
		currentDir := filepath.Join(tmpRuntimeDir, archiveName, dir)
		if fileutils.DirExists(currentDir) {
			tmpInstallDir = currentDir
		}
	}
	if tmpInstallDir == "" {
		// If no installDir was found assume the root of the archive
		tmpInstallDir = filepath.Join(tmpRuntimeDir, archiveName)
	}

	if err := fileutils.MoveAllFilesCrossDisk(tmpInstallDir, ci.runtimeDir); err != nil {
		logging.Error("moving files from %s after unpacking runtime: %v", tmpInstallDir, err)

		// It is possible that we get an Access Denied error (on Windows) while moving files to the installation directory.
		// Eg., https://rollbar.com/activestate/state-tool/items/297/occurrences/118875103987/
		// This might happen due to virus software or other access control software running on the user's machine,
		// and therefore we forward this information to the user.
		if os.IsPermission(err) {
			return locale.NewInputError("installer_err_runtime_move_files_access_denied", "", ci.runtimeDir, constants.ForumsURL)
		}
		return locale.WrapError(err, "installer_err_runtime_move_files_failed", "", tmpInstallDir, ci.runtimeDir)
	}

	tmpMetaFile := filepath.Join(tmpRuntimeDir, archiveName, "support", constants.RuntimeMetaFile)
	if fileutils.FileExists(tmpMetaFile) {
		target := filepath.Join(ci.runtimeDir, constants.RuntimeMetaFile)
		if err := fileutils.MkdirUnlessExists(filepath.Dir(target)); err != nil {
			return err
		}
		if err := os.Rename(tmpMetaFile, target); err != nil {
			return errs.Wrap(err, "os.Rename failed")
		}
	}

	tmpRelocFile := filepath.Join(tmpRuntimeDir, archiveName, "support/reloc.txt")
	if fileutils.FileExists(tmpRelocFile) {
		target := filepath.Join(ci.runtimeDir, "support/reloc.txt")
		if err := fileutils.MkdirUnlessExists(filepath.Dir(target)); err != nil {
			return err
		}
		if err := os.Rename(tmpRelocFile, target); err != nil {
			return errs.Wrap(err, "rename %s:%s failed", tmpRelocFile, target)
		}
	}

	if err := os.RemoveAll(tmpRuntimeDir); err != nil {
		logging.Error("removing %s after unpacking runtime: %v", tmpRuntimeDir, err)
		return locale.WrapError(err, "installer_err_runtime_rm_installdir", "", tmpRuntimeDir)
	}

	metaData, err := InitMetaData(ci.runtimeDir)
	if err != nil {
		return err
	}

	if err = Relocate(metaData, cb); err != nil {
		return err
	}

	if metaData.hasBinaryFile(constants.ActivePerlExecutable) {
		err := installPPMShim(filepath.Join(metaData.Path, metaData.BinaryLocations[0].Path))
		if err != nil {
			return locale.WrapError(err, "ppm_install_err")
		}
	}

	ci.env = ci.appendEnv(ci.env, metaData)

	return nil
}

func (ci *CamelInstall) appendEnv(env map[string]string, meta *MetaData) map[string]string {
	// Unset AffectedEnv
	if meta.AffectedEnv != "" {
		env[meta.AffectedEnv] = deleteMarker
	}

	for k, v := range meta.Env {
		env[k] = v
	}

	// Set up PATH according to binary locations
	for _, v := range meta.BinaryLocations {
		path := v.Path
		if v.Relative {
			path = filepath.Join(ci.runtimeDir, path)
		}
		env["PATH"] = prependPath(env["PATH"], path)
	}

	// Add DLL dir to PATH on Windows
	if meta.RelocationTargetBinaries != "" && rt.GOOS == "windows" {
		env["PATH"] = prependPath(env["PATH"], filepath.Join(ci.runtimeDir, meta.RelocationTargetBinaries))
	}

	return env
}

// Relocate will look through all of the files in this installation and replace any
// character sequence in those files containing the given prefix.
func Relocate(metaData *MetaData, cb func()) error {
	prefix := metaData.RelocationDir

	for _, tr := range metaData.TargetedRelocations {
		path := filepath.Join(metaData.Path, tr.InDir)
		err := fileutils.ReplaceAllInDirectory(path, tr.SearchString, tr.Replacement,
			// only replace text files for now
			func(_ string, fileBytes []byte) bool {
				return !fileutils.IsBinary(fileBytes)
			})
		if err != nil {
			return errs.Wrap(err, "ReplaceAllInDirectory (Relocate) %s - %s:%s failed", path, tr.SearchString, tr.Replacement)
		}
	}

	if len(prefix) == 0 || prefix == metaData.Path {
		return nil
	}
	logging.Debug("relocating '%s' to '%s'", prefix, metaData.Path)
	binariesSeparate := rt.GOOS == "linux" && metaData.RelocationTargetBinaries != ""

	relocFilePath := filepath.Join(metaData.Path, "support", "reloc.txt")
	relocMap := map[string]bool{}
	if fileutils.FileExists(relocFilePath) {
		relocMap = loadRelocationFile(relocFilePath)
	}

	// Replace plain text files
	err := fileutils.ReplaceAllInDirectory(metaData.Path, prefix, metaData.Path,
		// Check if we want to include this
		func(p string, contents []byte) bool {
			suffix := strings.TrimPrefix(p, metaData.Path)
			if relocMap[suffix] {
				return true
			}
			if !strings.HasSuffix(p, filepath.FromSlash(constants.RuntimeMetaFile)) && (!binariesSeparate || !fileutils.IsBinary(contents)) {
				cb()
				return true
			}
			return false
		})
	if err != nil {
		return errs.Wrap(err, "ReplaceAllInDirectory (plain text) %s - %s:%s failed", metaData.Path, prefix, metaData.Path)
	}

	if binariesSeparate {
		replacement := filepath.Join(metaData.Path, metaData.RelocationTargetBinaries)
		// Replace binary files
		err = fileutils.ReplaceAllInDirectory(metaData.Path, prefix, replacement,
			// Binaries only
			func(p string, contents []byte) bool {
				if fileutils.IsBinary(contents) {
					cb()
					return true
				}
				return false
			})

		if err != nil {
			return errs.Wrap(err, "ReplaceAllInDirectory (binaries) %s - %s:%s failed", metaData.Path, prefix, replacement)
		}
	}

	return nil
}

// GetEnv returns the environment that is needed to use the installed runtime
func (ce *CamelEnv) GetEnv(inherit bool, projectDir string) (map[string]string, error) {
	var env map[string]string

	envData, err := fileutils.ReadFile(filepath.Join(ce.runtimeDir, envFile))
	if err != nil {
		return env, errs.Wrap(err, "Could not read "+envFile)
	}

	if err := json.Unmarshal(envData, &env); err != nil {
		return env, errs.Wrap(err, "Could not unmarshal "+envFile)
	}

	if inherit {
		env["PATH"] = prependPath(os.Getenv("PATH"), env["PATH"])
	}

	templateMeta := struct {
		ProjectDir string
	}{projectDir}

	resultEnv := map[string]string{}
	for k, v := range env {
		if v == deleteMarker {
			continue
		}

		// Dirty workaround until https://www.pivotaltracker.com/story/show/172033094 is implemented
		// This avoids projectDir dependant env vars from being written
		if projectDir == "" && strings.Contains(v, "ProjectDir") {
			continue
		}

		valueTemplate, err := template.New(k).Parse(v)
		if err != nil {
			logging.Error("Skipping env value with invalid value: %s:%s, error: %v", k, v, err)
			continue
		}
		var realValue bytes.Buffer
		err = valueTemplate.Execute(&realValue, templateMeta)
		if err != nil {
			logging.Error("Skipping env value whose value could not be parsed: %s:%s, error: %v", k, v, err)
			continue
		}
		resultEnv[k] = realValue.String()
	}
	return resultEnv, nil
}

// PostInstall creates completion markers for all artifact directories
func (ci *CamelInstall) PostInstall() error {
	env, err := json.Marshal(ci.env)
	if err != nil {
		return errs.Wrap(err, "Could not marshal camel environment")
	}

	if err := fileutils.WriteFile(filepath.Join(ci.runtimeDir, envFile), env); err != nil {
		return errs.Wrap(err, "Could not write "+envFile)
	}

	return nil
}

// IsInstalled checks if completion marker files exist for all artifacts
func (ci *CamelInstall) IsInstalled() bool {
	marker := filepath.Join(ci.runtimeDir, constants.RuntimeInstallationCompleteMarker)
	if !fileutils.FileExists(marker) {
		return false
	}

	contents, err := fileutils.ReadFile(marker)
	if err != nil {
		logging.Error("Could not read marker: %v", err)
		return false
	}

	return string(contents) == ci.commitID.String()
}

func prependPath(PATH, prefix string) string {
	var suffix string
	if PATH != "" {
		suffix = string(os.PathListSeparator) + PATH
	}
	return prefix + suffix
}
