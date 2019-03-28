package runtime

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// ActivePythonInstaller is an Installer for ActivePython runtimes.
type ActivePythonInstaller struct {
	installDir        string
	runtimeDownloader Downloader
	onDownload        func()
	onInstall         func()
}

// InitActivePythonInstaller creates a new ActivePythonInstaller after verifying the provided install-dir
// exists as a directory or can be created.
func InitActivePythonInstaller(installDir string) (Installer, *failures.Failure) {
	return NewActivePythonInstaller(installDir, InitRuntimeDownload(installDir))
}

// NewActivePythonInstaller creates a new ActivePythonInstaller after verifying the provided install-dir
// exists as a directory or can be created.
func NewActivePythonInstaller(installDir string, downloader Downloader) (Installer, *failures.Failure) {
	if fileutils.FileExists(installDir) {
		// install-dir exists, but is a regular file
		return nil, FailInstallDirInvalid.New("installer_err_installdir_isfile", installDir)
	} else if !fileutils.DirExists(installDir) {
		// make install-dir if does not exist
		if failure := fileutils.Mkdir(installDir); failure != nil {
			return nil, failure
		}
	}

	return &ActivePythonInstaller{
		installDir:        installDir,
		runtimeDownloader: downloader,
	}, nil
}

// InstallDir is the directory where this runtime will install to.
func (installer *ActivePythonInstaller) InstallDir() string {
	return installer.installDir
}

// Install will download the installer archive and invoke InstallFromArchive
func (installer *ActivePythonInstaller) Install() *failures.Failure {
	if installer.onDownload != nil {
		installer.onDownload()
	}
	archivePath, failure := installer.runtimeDownloader.Download()
	if failure != nil {
		return failure
	}
	archivePath = path.Join(installer.InstallDir(), archivePath)

	return installer.InstallFromArchive(archivePath)
}

// InstallFromArchive will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePython runtime to the configured runtime dir. Any failures
// during this process will result in a failed installation and the install-dir being removed.
func (installer *ActivePythonInstaller) InstallFromArchive(archivePath string) *failures.Failure {
	if installer.onInstall != nil {
		installer.onInstall()
	}

	runtimeName := apyRuntimeName(archivePath)

	if failure := installer.unpackRuntime(runtimeName, archivePath); failure != nil {
		removeInstallDir(installer.installDir)
		return failure
	}

	python, failure := installer.locatePythonExecutable(archivePath)
	if failure != nil {
		removeInstallDir(installer.InstallDir())
		return failure
	}

	// get prefixes for relocation
	prefixes, failure := installer.extractRelocationPrefixes(runtimeName, python)
	if failure != nil {
		removeInstallDir(installer.InstallDir())
		return failure
	}

	// relocate python
	if failure = installer.relocatePathPrefixes(prefixes); failure != nil {
		removeInstallDir(installer.InstallDir())
		return failure

	}
	return nil
}

// OnDownload registers a function to be called when a download occurs
func (installer *ActivePythonInstaller) OnDownload(f func()) { installer.onDownload = f }

// OnInstall registers a function to be called when an install occurs
func (installer *ActivePythonInstaller) OnInstall(f func()) { installer.onInstall = f }

// unpackRuntime will extract the `RuntimeName/INSTALLDIR` directory from the runtime archive
// to the configured installation dir. It will then move all files from install-dir/INSTALLDIR to
// its parent (install-dir) and finally remove install-dir/INSTALLDIR.
func (installer *ActivePythonInstaller) unpackRuntime(runtimeName, archivePath string) *failures.Failure {
	if failure := validateArchiveTarGz(archivePath); failure != nil {
		return failure
	}

	tmpRuntimeDir := path.Join(installer.installDir, runtimeName)
	tmpInstallDir := path.Join(tmpRuntimeDir, constants.ActivePythonInstallDir)
	err := archiver.DefaultTarGz.Unarchive(archivePath,
		installer.installDir)
	if err != nil {
		return FailArchiveInvalid.Wrap(err)
	}

	if !fileutils.DirExists(tmpInstallDir) {
		return FailRuntimeInvalid.New("installer_err_runtime_missing_install_dir", archivePath,
			path.Join(runtimeName, constants.ActivePythonInstallDir))
	}

	if failure := fileutils.MoveAllFiles(tmpInstallDir, installer.installDir); failure != nil {
		logging.Error("moving files from %s after unpacking runtime: %v", tmpInstallDir, failure.ToError())
		return FailRuntimeInstallation.New("installer_err_runtime_move_files_failed", tmpInstallDir)
	}

	if err = os.RemoveAll(tmpRuntimeDir); err != nil {
		logging.Error("removing %s after unpacking runtime: %v", tmpRuntimeDir, err)
		return FailRuntimeInstallation.New("installer_err_runtime_rm_installdir", tmpRuntimeDir)
	}

	return nil
}

// locatePythonExecutable will locate the path to the python binary in the runtime dir.
func (installer *ActivePythonInstaller) locatePythonExecutable(archivePath string) (string, *failures.Failure) {
	python2 := path.Join(installer.InstallDir(), "bin", constants.ActivePython2Executable)
	python3 := path.Join(installer.InstallDir(), "bin", constants.ActivePython3Executable)

	var executable string
	var executablePath string
	if fileutils.FileExists(python3) {
		executable = constants.ActivePython3Executable
		executablePath = python3
	} else if fileutils.FileExists(python2) {
		executable = constants.ActivePython2Executable
		executablePath = python2
	} else {
		return "", FailRuntimeInvalid.New("installer_err_runtime_no_executable", archivePath, constants.ActivePython2Executable, constants.ActivePython3Executable)
	}

	if !fileutils.IsExecutable(executablePath) {
		return "", FailRuntimeInvalid.New("installer_err_runtime_executable_not_exec", archivePath, executable)
	}
	return executablePath, nil
}

// extractRelocationPrefixes will extract the prefixes that need to be replaced in a relocation
// for this installation.
func (installer *ActivePythonInstaller) extractRelocationPrefixes(runtimeName string, python string) ([]string, *failures.Failure) {
	prefixBytes, err := exec.Command(python, "-c", "import activestate; print('\\n'.join(activestate.prefixes))").Output()
	if err != nil {
		if _, isExitError := err.(*exec.ExitError); isExitError {
			logging.Errorf("obtaining relocation prefixes: %v : %s", err, string(prefixBytes))
			return nil, FailRuntimeInvalid.New("installer_err_fail_obtain_prefixes", runtimeName)
		}
		return nil, FailRuntimeInvalid.Wrap(err)
	}
	return strings.Split(string(prefixBytes), "\n"), nil
}

// relocatePathPrefixes will look through all of the files in this installation and replace any
// character sequence in those files containing any value from the the prefixes slice. Prefixes
// assumed to be a slice of paths of some sort.
func (installer *ActivePythonInstaller) relocatePathPrefixes(prefixes []string) *failures.Failure {
	for _, prefix := range prefixes {
		if len(prefix) > 0 && prefix != installer.InstallDir() {
			logging.Debug("relocating '%s' to '%s'", prefix, installer.InstallDir())
			err := fileutils.ReplaceAllInDirectory(installer.InstallDir(), prefix, installer.InstallDir())
			if err != nil {
				return FailRuntimeInstallation.Wrap(err)
			}
		}
	}
	return nil
}

// apyRuntimeName uses the filename of the archive to determine a qualified name of a runtime. The assumption
// is that the archive filename is something like:
//
// /path/to/ActivePython-3.5.4.3504-linux-x86_64-glibc-2.12-404899.tar.gz
//
// Thus, the runtime name would be: ActivePython-3.5.4.3504-linux-x86_64-glibc-2.12-404899
func apyRuntimeName(archivePath string) string {
	return strings.TrimSuffix(strings.TrimSuffix(filepath.Base(archivePath), ".tar.gz"), ".tgz")
}
