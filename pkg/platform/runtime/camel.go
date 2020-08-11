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
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/unarchiver"
)

var _ Assembler = &CamelRuntime{}

const envFile = "activestate.env.json"
const deleteMarker = "!#DELETE#!"

// CamelRuntime holds all the meta-data necessary to activate a runtime
// environment for a Camel build
type CamelRuntime struct {
	commitID   strfmt.UUID
	artifacts  []*HeadChefArtifact
	runtimeDir string
	env        map[string]string
}

// NewCamelRuntime returns a new camel runtime assembler
// It filters the provided artifact list for use-able artifacts
func NewCamelRuntime(commitID strfmt.UUID, artifacts []*HeadChefArtifact, cacheDir string) (*CamelRuntime, *failures.Failure) {
	cr := &CamelRuntime{commitID, []*HeadChefArtifact{}, cacheDir, map[string]string{}}

	for _, artf := range artifacts {
		// filter artifacts
		if artf.URI == "" {
			continue
		}

		filename := filepath.Base(artf.URI.String())
		if !strings.HasSuffix(filename, cr.InstallerExtension()) || strings.Contains(filename, InstallerTestsSubstr) {
			continue
		}

		cr.artifacts = append(cr.artifacts, artf)
	}
	if len(cr.artifacts) == 0 {
		return cr, FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
	}
	return cr, nil
}

// InstallerExtension returns the expected file extension for archive file names
// We expect .zip for Windows and .tar.gz otherwise
func (cr *CamelRuntime) InstallerExtension() string {
	if rt.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// Unarchiver initializes and returns an Unarchiver instance that is able to
// unpack the downloaded artifact archives.
func (cr *CamelRuntime) Unarchiver() unarchiver.Unarchiver {
	if rt.GOOS == "windows" {
		return unarchiver.NewZip()
	}
	return unarchiver.NewTarGz()
}

// BuildEngine always returns Camel
func (cr *CamelRuntime) BuildEngine() BuildEngine {
	return Camel
}

// DownloadDirectory returns the download directory for a given artifact
// Each artifact is downloaded into its own temporary directory
func (cr *CamelRuntime) DownloadDirectory(artf *HeadChefArtifact) (string, *failures.Failure) {
	downloadDir, err := ioutil.TempDir("", "state-runtime-downloader")
	if err != nil {
		return downloadDir, failures.FailIO.Wrap(err)
	}
	return downloadDir, nil
}

// ArtifactsToDownload returns the artifacts that we need to download for this project
// It filters out all artifacts for which the final installation directory does not include a completion marker yet
func (cr *CamelRuntime) ArtifactsToDownload() []*HeadChefArtifact {
	return cr.artifacts
}

// PreInstall does nothing for camel builds
func (cr *CamelRuntime) PreInstall() *failures.Failure {
	if fileutils.DirExists(cr.runtimeDir) {
		empty, fail := fileutils.IsEmptyDir(cr.runtimeDir)
		if fail != nil {
			logging.Error("Could not check if target runtime dir is empty, this could cause issues.. %v", fail)
		} else if !empty {
			logging.Debug("Removing existing runtime")
			if err := os.RemoveAll(cr.runtimeDir); err != nil {
				logging.Error("Could not empty out target runtime dir prior to install, this could cause issues.. %v", err)
			}
		}
	}
	return nil
}

// PreUnpackArtifact ensures that the final installation directory exists and is
// useable.
// Note:  It will remove a previous installation
func (cr *CamelRuntime) PreUnpackArtifact(artf *HeadChefArtifact) *failures.Failure {
	if fileutils.FileExists(cr.runtimeDir) {
		// install-dir exists, but is a regular file
		return FailInstallDirInvalid.New("installer_err_installdir_isfile", cr.runtimeDir)
	}

	if fileutils.DirExists(cr.runtimeDir) {
		// remove previous installation
		if err := os.RemoveAll(cr.runtimeDir); err != nil {
			return failures.FailOS.Wrap(err, "failed to remove spurious previous installation")
		}
	}

	if fail := fileutils.MkdirUnlessExists(cr.runtimeDir); fail != nil {
		return fail
	}

	return nil
}

// PostUnpackArtifact parses the metadata file, runs the Relocation function (if
// necessary) and moves the artifact to its final destination
func (cr *CamelRuntime) PostUnpackArtifact(artf *HeadChefArtifact, tmpRuntimeDir string, archivePath string, cb func()) *failures.Failure {
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

	if fail := fileutils.MoveAllFilesCrossDisk(tmpInstallDir, cr.runtimeDir); fail != nil {
		underlyingError := fail.ToError()
		logging.Error("moving files from %s after unpacking runtime: %v", tmpInstallDir, underlyingError)

		// It is possible that we get an Access Denied error (on Windows) while moving files to the installation directory.
		// Eg., https://rollbar.com/activestate/state-tool/items/297/occurrences/118875103987/
		// This might happen due to virus software or other access control software running on the user's machine,
		// and therefore we forward this information to the user.
		if os.IsPermission(underlyingError) {
			return FailRuntimeInstallation.New("installer_err_runtime_move_files_access_denied", cr.runtimeDir, constants.ForumsURL)
		}
		return FailRuntimeInstallation.New("installer_err_runtime_move_files_failed", tmpInstallDir, cr.runtimeDir)
	}

	tmpMetaFile := filepath.Join(tmpRuntimeDir, archiveName, constants.RuntimeMetaFile)
	if fileutils.FileExists(tmpMetaFile) {
		target := filepath.Join(cr.runtimeDir, constants.RuntimeMetaFile)
		if fail := fileutils.MkdirUnlessExists(filepath.Dir(target)); fail != nil {
			return fail
		}
		if err := os.Rename(tmpMetaFile, target); err != nil {
			return FailRuntimeInstallation.Wrap(err)
		}
	}

	tmpRelocFile := filepath.Join(tmpRuntimeDir, archiveName, "support/reloc.txt")
	if fileutils.FileExists(tmpRelocFile) {
		target := filepath.Join(cr.runtimeDir, "support/reloc.txt")
		if fail := fileutils.MkdirUnlessExists(filepath.Dir(target)); fail != nil {
			return fail
		}
		if err := os.Rename(tmpRelocFile, target); err != nil {
			return FailRuntimeInstallation.Wrap(err)
		}
	}

	if err := os.RemoveAll(tmpRuntimeDir); err != nil {
		logging.Error("removing %s after unpacking runtime: %v", tmpRuntimeDir, err)
		return FailRuntimeInstallation.New("installer_err_runtime_rm_installdir", tmpRuntimeDir)
	}

	metaData, fail := InitMetaData(cr.runtimeDir)
	if fail != nil {
		return fail
	}

	if fail = Relocate(metaData, cb); fail != nil {
		return fail
	}

	if metaData.hasBinaryFile(constants.ActivePerlExecutable) {
		err := installPPMShim(filepath.Join(metaData.Path, metaData.BinaryLocations[0].Path))
		if err != nil {
			return FailRuntimeInstallation.New("ppm_install_err")
		}
	}

	cr.env = cr.appendEnv(cr.env, metaData)

	return nil
}

func (cr *CamelRuntime) appendEnv(env map[string]string, meta *MetaData) map[string]string {
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
			path = filepath.Join(cr.runtimeDir, path)
		}
		env["PATH"] = prependPath(env["PATH"], path)
	}

	// Add DLL dir to PATH on Windows
	if meta.RelocationTargetBinaries != "" && rt.GOOS == "windows" {
		env["PATH"] = prependPath(env["PATH"], filepath.Join(cr.runtimeDir, meta.RelocationTargetBinaries))
	}

	return env
}

// Relocate will look through all of the files in this installation and replace any
// character sequence in those files containing the given prefix.
func Relocate(metaData *MetaData, cb func()) *failures.Failure {
	prefix := metaData.RelocationDir

	for _, tr := range metaData.TargetedRelocations {
		err := fileutils.ReplaceAllInDirectory(filepath.Join(metaData.Path, tr.InDir), tr.SearchString, tr.Replacement,
			// only replace text files for now
			func(_ string, fileBytes []byte) bool {
				return !fileutils.IsBinary(fileBytes)
			})
		if err != nil {
			return FailRuntimeInstallation.Wrap(err)
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
		return FailRuntimeInstallation.Wrap(err)
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
			return FailRuntimeInstallation.Wrap(err)
		}
	}

	return nil
}

// GetEnv returns the environment that is needed to use the installed runtime
func (cr *CamelRuntime) GetEnv(inherit bool, projectDir string) (map[string]string, error) {
	var env map[string]string

	envData, err := fileutils.ReadFile(filepath.Join(cr.runtimeDir, envFile))
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
func (cr *CamelRuntime) PostInstall() error {
	fail := fileutils.WriteFile(filepath.Join(cr.runtimeDir, constants.RuntimeInstallationCompleteMarker), []byte(cr.commitID.String()))
	if fail != nil {
		return errs.Wrap(fail, "could not set completion marker")
	}

	env, err := json.Marshal(cr.env)
	if err != nil {
		return errs.Wrap(err, "Could not marshal camel environment")
	}

	if fail := fileutils.WriteFile(filepath.Join(cr.runtimeDir, envFile), env); fail != nil {
		return errs.Wrap(fail, "Could not write "+envFile)
	}

	return nil
}

// IsInstalled checks if completion marker files exist for all artifacts
func (cr *CamelRuntime) IsInstalled() bool {
	marker := filepath.Join(cr.runtimeDir, constants.RuntimeInstallationCompleteMarker)
	if !fileutils.FileExists(marker) {
		return false
	}

	contents, fail := fileutils.ReadFile(marker)
	if fail != nil {
		logging.Error("Could not read marker: %v", fail)
		return false
	}

	return string(contents) == cr.commitID.String()
}

func prependPath(PATH, prefix string) string {
	var suffix string
	if PATH != "" {
		suffix = string(os.PathListSeparator) + PATH
	}
	return prefix + suffix
}
