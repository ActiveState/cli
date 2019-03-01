package installer

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mholt/archiver"
)

// ActivePythonInstaller is an Installer for ActivePython distributions.
type ActivePythonInstaller struct {
	installDir  string
	archivePath string
	distName    string
}

// apyDistName uses the filename of the archive to determine a qualified name of a distribution. The assumption
// is that the archive filename is something like:
//
// /path/to/ActivePython-3.5.4.3504-linux-x86_64-glibc-2.12-404899.tar.gz
//
// Thus, the distribution name would be: ActivePython-3.5.4.3504-linux-x86_64-glibc-2.12-404899
func apyDistName(archivePath string) string {
	return strings.TrimSuffix(strings.TrimSuffix(filepath.Base(archivePath), ".tar.gz"), ".tgz")
}

// func validate

// NewActivePythonInstaller creates a new ActivePythonInstaller after verifying the following:
//
// 1. the provided install-dir exists as a directory or can be created
// 2. the provided installer archive exists and is named with .tar.gz or .tgz
// 3. that a distribution is not already installed in the install-dir
func NewActivePythonInstaller(installDir, installerArchivePath string) (*ActivePythonInstaller, *failures.Failure) {
	if fileutils.FileExists(installDir) {
		// install-dir exists, but is a regular file
		return nil, FailInstallDirInvalid.New("installer_err_installdir_isfile", installDir)
	} else if !fileutils.DirExists(installDir) {
		// make install-dir if does not exist
		if failure := fileutils.Mkdir(installDir); failure != nil {
			return nil, failure
		}
	} else if !fileutils.FileExists(installerArchivePath) {
		return nil, FailArchiveInvalid.New("installer_err_archive_notfound", installerArchivePath)
	} else if archiver.DefaultTarGz.CheckExt(installerArchivePath) != nil {
		return nil, FailArchiveInvalid.New("installer_err_archive_badext", installerArchivePath)
	} else if isEmpty, failure := fileutils.IsEmptyDir(installDir); !isEmpty || failure != nil {
		if failure != nil {
			logging.Error("reading files in directory '%s': %v", installDir, failure.ToError())
		}
		return nil, FailDistInstallation.New("installer_err_dist_already_exists", installDir)
	}

	distName := apyDistName(installerArchivePath)

	return &ActivePythonInstaller{
		distName:    distName,
		installDir:  installDir,
		archivePath: installerArchivePath,
	}, nil
}

// DistributionName is the qualified name of the distribution to install.
func (installer *ActivePythonInstaller) DistributionName() string {
	return installer.distName
}

// InstallDir is the directory where this distribution will install to.
func (installer *ActivePythonInstaller) InstallDir() string {
	return installer.installDir
}

// ArchivePath is the path to the installer archive.
func (installer *ActivePythonInstaller) ArchivePath() string {
	return installer.archivePath
}

// Install will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePython distribution to the configured distribution dir. Any failures
// during this process will result in a failed installation and the install-dir being removed.
func (installer *ActivePythonInstaller) Install() *failures.Failure {
	if failure := installer.unpackDist(); failure != nil {
		removeInstallDir(installer.installDir)
		return failure
	}

	python, failure := installer.locatePythonExecutable()
	if failure != nil {
		removeInstallDir(installer.InstallDir())
		return failure
	}

	// get prefixes for relocation
	prefixes, failure := installer.extractRelocationPrefixes(python)
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

// unpackDist will extract the `DistributionName/INSTALLDIR` directory from the distribution archive
// to the configured installation dir. It will then move all files from install-dir/INSTALLDIR to
// its parent (install-dir) and finally remove install-dir/INSTALLDIR.
func (installer *ActivePythonInstaller) unpackDist() *failures.Failure {
	tmpInstallDir := path.Join(installer.installDir, constants.ActivePythonInstallDir)

	err := archiver.DefaultTarGz.Extract(installer.ArchivePath(),
		path.Join(installer.DistributionName(), constants.ActivePythonInstallDir),
		installer.installDir)
	if err != nil {
		return FailArchiveInvalid.Wrap(err)
	}

	if !fileutils.DirExists(tmpInstallDir) {
		return FailDistInvalid.New("installer_err_dist_missing_install_dir", installer.ArchivePath(),
			path.Join(installer.DistributionName(), constants.ActivePythonInstallDir))
	}

	if err := moveFiles(tmpInstallDir, installer.installDir); err != nil {
		logging.Error("moving files from %s after unpacking distribution: %v", tmpInstallDir, err)
		return FailDistInstallation.New("installer_err_dist_move_files_failed", tmpInstallDir)
	}

	if err = os.RemoveAll(tmpInstallDir); err != nil {
		logging.Error("removing %s after unpacking distribution: %v", tmpInstallDir, err)
		return FailDistInstallation.New("installer_err_dist_rm_installdir", tmpInstallDir)
	}

	return nil
}

// locatePythonExecutable will locate the path to the python binary in the distribution dir.
func (installer *ActivePythonInstaller) locatePythonExecutable() (string, *failures.Failure) {
	python3 := path.Join(installer.InstallDir(), "bin", constants.ActivePythonExecutable)
	if !fileutils.FileExists(python3) {
		return "", FailDistInvalid.New("installer_err_dist_no_executable", installer.ArchivePath(), constants.ActivePythonExecutable)
	} else if !fileutils.IsExecutable(python3) {
		return "", FailDistInvalid.New("installer_err_dist_executable_not_exec", installer.ArchivePath(), constants.ActivePythonExecutable)
	}
	return python3, nil
}

// extractRelocationPrefixes will extract the prefixes that need to be replaced in a relocation
// for this installation.
func (installer *ActivePythonInstaller) extractRelocationPrefixes(python string) ([]string, *failures.Failure) {
	prefixBytes, err := exec.Command(python, "-c", "import activestate; print(*activestate.prefixes, sep='\\n')").Output()
	if err != nil {
		if _, isExitError := err.(*exec.ExitError); isExitError {
			logging.Errorf("obtaining relocation prefixes: %v : %s", err, string(prefixBytes))
			return nil, FailDistInvalid.New("installer_err_fail_obtain_prefixes", installer.DistributionName())
		}
		return nil, FailDistInvalid.Wrap(err)
	}
	return strings.Split(string(prefixBytes), "\n"), nil
}

// relocatePathPrefixes will look through all of the files in this installation and replace any
// character sequence in those files containing any value from the the prefixes slice. Prefixes
// assumed to be a slice of paths of some sort.
func (installer *ActivePythonInstaller) relocatePathPrefixes(prefixes []string) *failures.Failure {
	for _, prefix := range prefixes {
		if len(prefix) > 0 && prefix != installer.InstallDir() {
			err := fileutils.ReplaceAllInDirectory(installer.InstallDir(), prefix, installer.InstallDir())
			if err != nil {
				return FailDistInstallation.Wrap(err)
			}
		}
	}
	return nil
}

// removeInstallDir will remove a given directory and log any errors resulting from
// that removal. No errors are returned.
func removeInstallDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		logging.Errorf("attempting to remove install dir '%s': %v", dir, err)
	}
}

// moveFiles will move all of the files/dirs from one directory to another.
func moveFiles(fromPath, toPath string) error {
	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return err
	}

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		err := os.Rename(path.Join(fromPath, fileInfo.Name()), path.Join(toPath, fileInfo.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
