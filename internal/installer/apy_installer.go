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
	distDir     string
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

// NewActivePythonInstaller creates a new ActivePythonInstaller after verifying the following:
//
// 1. the provided install-dir (e.g. a virtualenvironment dir) exists
// 2. the provided installer archive exists and is named with .tar.gz or .tgz
// 3. that a distribution with the same qualified-name is not already installed
func NewActivePythonInstaller(installDir, installerArchivePath string) (*ActivePythonInstaller, *failures.Failure) {
	if !fileutils.DirExists(installDir) {
		return nil, FailWorkingDirInvalid.New("installer_err_workingdir_invalid", installDir)
	} else if !fileutils.FileExists(installerArchivePath) {
		return nil, FailArchiveInvalid.New("installer_err_archive_notfound", installerArchivePath)
	} else if archiver.DefaultTarGz.CheckExt(installerArchivePath) != nil {
		return nil, FailArchiveInvalid.New("installer_err_archive_badext", installerArchivePath)
	}

	distName := apyDistName(installerArchivePath)
	distDir := path.Join(installDir, constants.ActivePythonDistsDir, distName)

	if fileutils.DirExists(distDir) {
		return nil, FailDistInstallation.New("installer_err_dist_already_exists", distName)
	}

	return &ActivePythonInstaller{
		distName:    distName,
		distDir:     distDir,
		archivePath: installerArchivePath,
	}, nil
}

// DistributionName is the qualified name of the distribution to install.
func (installer *ActivePythonInstaller) DistributionName() string {
	return installer.distName
}

// DistributionDir is the directory where this distribution will install to.
func (installer *ActivePythonInstaller) DistributionDir() string {
	return installer.distDir
}

// ArchivePath is the path to the installer archive.
func (installer *ActivePythonInstaller) ArchivePath() string {
	return installer.archivePath
}

// Install will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePython distribution to the configured distribution dir.
func (installer *ActivePythonInstaller) Install() *failures.Failure {
	if failure := installer.unpackDist(); failure != nil {
		return failure
	}

	python, failure := installer.locatePythonExecutable()
	if failure != nil {
		removeDistributionDir(installer.DistributionDir())
		return failure
	}

	// get prefixes for relocation
	prefixes, failure := installer.extractRelocationPrefixes(python)
	if failure != nil {
		removeDistributionDir(installer.DistributionDir())
		return failure
	}

	// relocate python
	if failure = installer.relocatePathPrefixes(prefixes); failure != nil {
		removeDistributionDir(installer.DistributionDir())
		return failure

	}
	return nil
}

// unpackDist will extract the `DistributionName/INSTALLDIR` directory from the distribution archive
// to the parent dir of `DistributionDir`, and then rename INSTALLDIR to the value of `DistributionName`.
func (installer *ActivePythonInstaller) unpackDist() *failures.Failure {
	installDirParent := path.Dir(installer.DistributionDir())
	installDir := path.Join(installDirParent, constants.ActivePythonInstallDir)

	err := archiver.DefaultTarGz.Extract(installer.ArchivePath(),
		path.Join(installer.DistributionName(), constants.ActivePythonInstallDir),
		installDirParent)
	if err != nil {
		return FailArchiveInvalid.Wrap(err)
	}

	if !fileutils.DirExists(installDir) {
		return FailDistInvalid.New("installer_err_dist_missing_install_dir", installer.ArchivePath(),
			path.Join(installer.DistributionName(), constants.ActivePythonInstallDir))
	}

	err = os.Rename(installDir, installer.DistributionDir())
	if err != nil {
		os.RemoveAll(installDir)
		return FailDistInvalid.Wrap(err)
	}

	return nil
}

// locatePythonExecutable will locate the path to the python binary in the distribution dir.
func (installer *ActivePythonInstaller) locatePythonExecutable() (string, *failures.Failure) {
	python3 := path.Join(installer.DistributionDir(), "bin", constants.ActivePythonExecutable)
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
		if len(prefix) > 0 && prefix != installer.DistributionDir() {
			err := fileutils.ReplaceAllInDirectory(installer.DistributionDir(), prefix, installer.DistributionDir())
			if err != nil {
				return FailDistInstallation.Wrap(err)
			}
		}
	}
	return nil
}

// removeDistributionDir will remove a given directory and log any errors resulting from
// that removal. No errors are returned.
func removeDistributionDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		logging.Errorf("attempting to remove distribution dir '%s': %v", dir, err)
	}
}
