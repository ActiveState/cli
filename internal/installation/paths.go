package installation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

const (
	// CfgInstallPath is the configuration key for the path where the State Tool is installed
	CfgInstallPath = "installation_path"

	// CfgTransitionalStateToolPath is the configuration key for the path where a transitional State Tool might still be stored
	CfgTransitionalStateToolPath = "transitional_installation_path"

	BinDirName = "bin"

	InstallDirMarker = ".state_install_root"
)

func DefaultInstallPath() (string, error) {
	return InstallPathForBranch(constants.BranchName)
}

func InstallPath() (string, error) {
	// Facilitate use-case of running executables from the build dir while developing
	if !condition.BuiltViaCI() && strings.Contains(os.Args[0], "/build/") {
		return filepath.Dir(os.Args[0]), nil
	}
	if path, ok := os.LookupEnv(constants.OverwriteDefaultInstallationPathEnvVarName); ok {
		return path, nil
	}

	// If State Tool is already exists then we should detect the install path from there
	stateInfo := appinfo.StateApp()
	activeStateOwnedPath := strings.Contains(strings.ToLower(stateInfo.Exec()), "activestate")
	installRootFile := filepath.Join(filepath.Dir(stateInfo.Exec()), InstallDirMarker)
	if fileutils.TargetExists(stateInfo.Exec()) && fileutils.FileExists(installRootFile) && activeStateOwnedPath {
		return filepath.Dir(filepath.Dir(stateInfo.Exec())), nil // <return this>/bin/state.exe
	}

	return DefaultInstallPath()
}

func BinPathFromInstallPath(installPath string) (string, error) {
	if installPath == "" {
		var err error
		installPath, err = InstallPath()
		if err != nil {
			return installPath, errs.Wrap(err, "Could not detect InstallPath while searching for BinPath")
		}
	}

	return filepath.Join(installPath, BinDirName), nil
}

func InstalledOnPath(installRoot string) (bool, string, error) {
	if !fileutils.DirExists(installRoot) {
		return false, "", nil
	}

	binPath, err := BinPathFromInstallPath(installRoot)
	if err != nil {
		return false, "", errs.Wrap(err, "Could not detect binPath from BinPathFromInstallPath")
	}

	found, err := fileutils.FindFileInPath(binPath, InstallDirMarker)
	if err != nil {
		if errors.Is(err, fileutils.ErrorFileNotFound) {
			binPath = filepath.Dir(found)
		} else {
			return false, "", errs.Wrap(err, "Could not find install directory marker file")
		}
	}

	path := appinfo.StateApp(binPath).Exec()
	return fileutils.TargetExists(path), filepath.Dir(path), nil
}

func LauncherInstallPath() (string, error) {
	if path, ok := os.LookupEnv(constants.OverwriteDefaultSystemPathEnvVarName); ok {
		return path, nil
	}
	return defaultSystemInstallPath()
}

func IsInstallRoot(dir string) bool {
	return fileutils.FileExists(filepath.Join(dir, InstallDirMarker))
}
