package runtime

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/phayes/permbits"
)

var _ Assembler = &CamelRuntime{}

// CamelRuntime holds all the meta-data necessary to activate a runtime
// environment for a Camel build
type CamelRuntime struct {
	artifactMap map[string]*HeadChefArtifact
	cacheDir    string
	installDirs []string
}

// NewCamelRuntime returns a new camel runtime assembler
// It filters the provided artifact list for use-able artifacts
func NewCamelRuntime(artifacts []*HeadChefArtifact, cacheDir string) (*CamelRuntime, *failures.Failure) {
	artifactMap := map[string]*HeadChefArtifact{}

	cr := &CamelRuntime{cacheDir: cacheDir}

	for _, artf := range artifacts {
		// filter artifacts
		if artf.URI == "" {
			continue
		}

		filename := filepath.Base(artf.URI.String())
		if !strings.HasSuffix(filename, cr.InstallerExtension()) || strings.Contains(filename, InstallerTestsSubstr) {
			continue
		}
		installDir := cr.InstallationDirectory(artf)

		artifactMap[installDir] = artf
		cr.installDirs = append(cr.installDirs, installDir)
	}
	if len(artifactMap) == 0 {
		return cr, FailNoValidArtifact.New(locale.T("err_no_valid_artifact"))
	}
	cr.artifactMap = artifactMap
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

// InstallDirs returns the installation directories for the artifacts
func (cr *CamelRuntime) InstallDirs() []string {
	return cr.installDirs
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

// ArtifactsToDownloadAndUnpack returns the artifacts that we need to download for this project
// It filters out all artifacts for which the final installation directory does not include a completion marker yet
func (cr *CamelRuntime) ArtifactsToDownloadAndUnpack() ([]*HeadChefArtifact, map[string]*HeadChefArtifact) {
	downloadArtfs := []*HeadChefArtifact{}

	for installDir, artf := range cr.artifactMap {
		if !fileutils.FileExists(filepath.Join(installDir, constants.RuntimeInstallationCompleteMarker)) {
			downloadArtfs = append(downloadArtfs, artf)
		}
	}
	return downloadArtfs, map[string]*HeadChefArtifact{}
}

// InstallationDirectory returns the local directory into which the artifact files need to be unpacked
func (cr *CamelRuntime) InstallationDirectory(artf *HeadChefArtifact) string {

	installDir := filepath.Join(cr.cacheDir, hash.ShortHash(artf.ArtifactID.String()))

	return installDir
}

// PreInstall does nothing for camel builds
func (cr *CamelRuntime) PreInstall() *failures.Failure {
	return nil
}

// PreUnpackArtifact ensures that the final installation directory exists and is
// useable.
// Note:  It will remove a previous installation
func (cr *CamelRuntime) PreUnpackArtifact(artf *HeadChefArtifact) *failures.Failure {
	installDir := cr.InstallationDirectory(artf)

	if fileutils.FileExists(installDir) {
		// install-dir exists, but is a regular file
		return FailInstallDirInvalid.New("installer_err_installdir_isfile", installDir)
	}

	if fileutils.DirExists(installDir) {
		// remove previous installation
		if err := os.RemoveAll(installDir); err != nil {
			return failures.FailOS.Wrap(err, "failed to remove spurious previous installation")
		}
	}

	if fail := fileutils.MkdirUnlessExists(installDir); fail != nil {
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

	installDir := cr.InstallationDirectory(artf)

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

	if fail := fileutils.MoveAllFilesCrossDisk(tmpInstallDir, installDir); fail != nil {
		underlyingError := fail.ToError()
		logging.Error("moving files from %s after unpacking runtime: %v", tmpInstallDir, underlyingError)

		// It is possible that we get an Access Denied error (on Windows) while moving files to the installation directory.
		// Eg., https://rollbar.com/activestate/state-tool/items/297/occurrences/118875103987/
		// This might happen due to virus software or other access control software running on the user's machine,
		// and therefore we forward this information to the user.
		if os.IsPermission(underlyingError) {
			return FailRuntimeInstallation.New("installer_err_runtime_move_files_access_denied", installDir, constants.ForumsURL)
		}
		return FailRuntimeInstallation.New("installer_err_runtime_move_files_failed", tmpInstallDir, installDir)
	}

	tmpMetaFile := filepath.Join(tmpRuntimeDir, archiveName, constants.RuntimeMetaFile)
	if fileutils.FileExists(tmpMetaFile) {
		target := filepath.Join(installDir, constants.RuntimeMetaFile)
		if fail := fileutils.MkdirUnlessExists(filepath.Dir(target)); fail != nil {
			return fail
		}
		if err := os.Rename(tmpMetaFile, target); err != nil {
			return FailRuntimeInstallation.Wrap(err)
		}
	}

	tmpRelocFile := filepath.Join(tmpRuntimeDir, archiveName, "support/reloc.txt")
	if fileutils.FileExists(tmpRelocFile) {
		target := filepath.Join(installDir, "support/reloc.txt")
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

	metaData, fail := InitMetaData(installDir)
	if fail != nil {
		return fail
	}

	if fail = Relocate(metaData, cb); fail != nil {
		return fail
	}

	if metaData.hasBinaryFile(constants.ActivePerlExecutable) {
		err := installPPMShim(metaData)
		if err != nil {
			return FailRuntimeInstallation.New("ppm_install_err")
		}
	}

	return nil
}

func installPPMShim(metaData *MetaData) error {
	resp, err := http.Get(fmt.Sprintf("%s/ppm-%s", constants.PPMDownloadURL, runtime.GOOS))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	td := filepath.Join(metaData.Path, metaData.BinaryLocations[0].Path)
	ppmExe := "ppm"
	if runtime.GOOS == "windows" {
		ppmExe = "ppm.exe"
	}

	ppmTarget := filepath.Join(td, ppmExe)

	// remove old ppm command (if it existed before)
	_ = os.Remove(ppmTarget)

	out, err := os.Create(ppmTarget)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	out.Close()
	permissions, _ := permbits.Stat(ppmTarget)
	permissions.SetUserExecute(true)
	err = permbits.Chmod(ppmTarget, permissions)
	if err != nil {
		return err
	}
	return nil
}

// Relocate will look through all of the files in this installation and replace any
// character sequence in those files containing the given prefix.
func Relocate(metaData *MetaData, cb func()) *failures.Failure {
	prefix := metaData.RelocationDir

	for _, tr := range metaData.TargetedRelocations {
		err := fileutils.ReplaceAllInDirectory(tr.InDir, tr.SearchString, tr.Replacement,
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
			if !strings.HasSuffix(p, constants.RuntimeMetaFile) && (!binariesSeparate || !fileutils.IsBinary(contents)) {
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
	env := map[string]string{"PATH": ""}
	if inherit {
		env["PATH"] = os.Getenv("PATH")
	}

	if len(cr.installDirs) == 0 {
		return nil, locale.NewError("err_requires_runtime_download", "You need to download the runtime environment before you can use it")
	}

	for _, artifactPath := range cr.installDirs {
		meta, fail := InitMetaData(artifactPath)
		if fail != nil {
			return nil, locale.WrapError(
				fail,
				"err_get_env_metadata_error",
				"Your installation or build is corrupted.  Try re-installing the project, or update your build.  If the problem persists, please report the issue on our forums: {{.V0}}",
				constants.ForumsURL,
			)
		}

		// Unset AffectedEnv
		if meta.AffectedEnv != "" {
			delete(env, meta.AffectedEnv)
		}

		// Set up env according to artifact meta
		templateMeta := struct {
			RelocationDir string
			ProjectDir    string
		}{"", projectDir}
		for k, v := range meta.Env {
			// Dirty workaround until https://www.pivotaltracker.com/story/show/172033094 is implemented
			// This avoids projectDir dependant env vars from being written
			if projectDir == "" && strings.Contains(v, "ProjectDir") {
				continue
			}

			// XXX: This will replace the RelocationDir string with the funky string that camel introduces during build time.
			// We probably want: templateMeta.RelocationDir = artifactPath
			// BUT: From what I know there is no metadata file that actually uses this feature.
			// And as we have seen before, people do not like to do changes to camel.
			// It is and most likely will never be used.
			templateMeta.RelocationDir = meta.RelocationDir
			valueTemplate, err := template.New(k).Parse(v)
			if err != nil {
				logging.Error("Skipping artifact with invalid value: %s:%s, error: %v", k, v, err)
				continue
			}
			var realValue bytes.Buffer
			err = valueTemplate.Execute(&realValue, templateMeta)
			if err != nil {
				logging.Error("Skipping artifact whose value could not be parsed: %s:%s, error: %v", k, v, err)
				continue
			}
			env[k] = realValue.String()
		}

		// Set up PATH according to binary locations
		for _, v := range meta.BinaryLocations {
			path := v.Path
			if v.Relative {
				path = filepath.Join(artifactPath, path)
			}
			env["PATH"] = prependPath(env["PATH"], path)
		}

		// Add DLL dir to PATH on Windows
		if meta.RelocationTargetBinaries != "" && rt.GOOS == "windows" {
			env["PATH"] = prependPath(env["PATH"], filepath.Join(meta.Path, meta.RelocationTargetBinaries))
		}
	}
	return env, nil
}

// PostInstall creates completion markers for all artifact directories
func (cr *CamelRuntime) PostInstall() error {
	for _, instDir := range cr.installDirs {
		fail := fileutils.Touch(filepath.Join(instDir, constants.RuntimeInstallationCompleteMarker))
		if fail != nil {
			return errs.Wrap(fail, "could not set completion marker")
		}
	}
	return nil
}

// IsInstalled checks if completion marker files exist for all artifacts
func (cr *CamelRuntime) IsInstalled() bool {
	for _, instDir := range cr.installDirs {
		if !fileutils.FileExists(filepath.Join(instDir, constants.RuntimeInstallationCompleteMarker)) {
			return false
		}
	}
	return true
}

func prependPath(PATH, prefix string) string {
	var suffix string
	if PATH != "" {
		suffix = string(os.PathListSeparator) + PATH
	}
	return prefix + suffix
}
