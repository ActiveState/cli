package installer

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mholt/archiver"
)

// ActivePythonDistsDir represents the base name a the directory where apy dists will be installed under.
const ActivePythonDistsDir = "python"

// ActivePythonInstallScript represents the canonical apy installer script name.
const ActivePythonInstallScript = "install.sh"

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
// 1. the provided working-dir (e.g. a virtualenvironment dir) exists
// 2. the provided installer archive exists and is named with .tar.gz or .tgz
func NewActivePythonInstaller(workingDir, installerArchivePath string) (*ActivePythonInstaller, *failures.Failure) {
	if !fileutils.DirExists(workingDir) {
		return nil, FailWorkingDirInvalid.New("installer_err_workingdir_invalid", workingDir)
	} else if !fileutils.FileExists(installerArchivePath) {
		return nil, FailArchiveInvalid.New("installer_err_archive_notfound", installerArchivePath)
	} else if !archiver.TarGz.Match(installerArchivePath) {
		return nil, FailArchiveInvalid.New("installer_err_archive_badext", installerArchivePath)
	}

	distName := apyDistName(installerArchivePath)
	return &ActivePythonInstaller{
		distName:    distName,
		distDir:     path.Join(workingDir, ActivePythonDistsDir, distName),
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
	installerDir, failure := installer.unpackInstaller()
	if failure != nil {
		return failure
	}
	defer os.RemoveAll(installerDir)

	installScript, failure := installer.locateInstallScript(installerDir)
	if failure != nil {
		return failure
	}

	// prep distribution directory
	if failure := fileutils.MkdirUnlessExists(installer.DistributionDir()); failure != nil {
		return failure
	}

	// run the installer
	if failure := installer.execInstallerScript(installScript); failure != nil {
		os.RemoveAll(installer.DistributionDir())
		return failure
	}

	return nil
}

// unpackInstaller will create a temporary directory to unpack the installer archive's tarball to. It
// will then attempt to unpack the installer archive to the temp-dir. If successful, the path to the
// temp-dir is returned; otherwise the failure is. Upon success, you will then need to remove the
// temp-dir yourself.
func (installer *ActivePythonInstaller) unpackInstaller() (string, *failures.Failure) {
	installerDir, err := ioutil.TempDir("", installer.DistributionName())
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	} else if err := archiver.TarGz.Open(installer.ArchivePath(), installerDir); err != nil {
		os.RemoveAll(installerDir)
		return "", FailArchiveInvalid.Wrap(err)
	}
	return installerDir, nil
}

// locateInstallScript will locate the path to an ActivePython installer script in the unpacked installer archive.
func (installer *ActivePythonInstaller) locateInstallScript(installerDir string) (string, *failures.Failure) {
	installScriptDir := path.Join(installerDir, installer.DistributionName())
	installScript := path.Join(installScriptDir, ActivePythonInstallScript)
	if !fileutils.DirExists(installScriptDir) {
		return "", FailDistInvalid.New("installer_err_dist_missing_root_dir", installer.ArchivePath(), installer.DistributionName())
	} else if !fileutils.FileExists(installScript) {
		return "", FailDistInvalid.New("installer_err_dist_no_install_script", installer.ArchivePath(), ActivePythonInstallScript)
	} else if !fileutils.IsExecutable(installScript) {
		return "", FailDistInvalid.New("installer_err_dist_install_script_no_exec", installer.ArchivePath(), ActivePythonInstallScript)
	}
	return installScript, nil
}

func (installer *ActivePythonInstaller) execInstallerScript(installScript string) *failures.Failure {
	// apy installer tarballs come with an install.sh that accepts a "-I <install-dir>" flag
	installCmd := exec.Command(installScript, "-I", installer.DistributionDir())
	installCmd.Stdin, installCmd.Stdout, installCmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := installCmd.Run(); err != nil {
		if _, isExitError := err.(*exec.ExitError); isExitError {
			return FailDistInstallation.New("installer_err_installscript_failed", installer.DistributionName(), err.Error())
		}
		return FailDistInstallation.Wrap(err)
	}

	return nil
}
