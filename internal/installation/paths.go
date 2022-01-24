package installation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

// CfgInstallPath is the configuration key for the path where the State Tool is installed
const CfgInstallPath = "installation_path"

// CfgTransitionalStateToolPath is the configuration key for the path where a transitional State Tool might still be stored
const CfgTransitionalStateToolPath = "transitional_installation_path"

const BinDirName = "bin"

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
	if fileutils.TargetExists(stateInfo.Exec()) {
		if filepath.Base(filepath.Dir(stateInfo.Exec())) == BinDirName && activeStateOwnedPath {
			return filepath.Dir(filepath.Dir(stateInfo.Exec())), nil // <return this>/bin/state.exe
		}
		return filepath.Dir(stateInfo.Exec()), nil // <return this>/state.exe
	}

	return DefaultInstallPath()
}

func BinPath() (string, error) {
	return BinPathFromInstallPath("")
}

func BinPathFromInstallPath(installPath string) (string, error) {
	if installPath == "" {
		var err error
		installPath, err = InstallPath()
		if err != nil {
			return installPath, errs.Wrap(err, "Could not detect InstallPath while searching for BinPath")
		}
	}

	binDir := filepath.Join(installPath, BinDirName)
	if fileutils.FileExists(binDir) {
		return binDir, nil
	}

	return installPath, nil
}

func InstalledOnPath(installPath string) (bool, string, error) {
	binPath, err := BinPathFromInstallPath(installPath)
	if err != nil {
		return false, "", errs.Wrap(err, "Could not detect binPath from BinPathFromInstallPath")
	}
	path := appinfo.StateApp(binPath).Exec()
	return fileutils.TargetExists(path), path, nil
}

func LauncherInstallPath() (string, error) {
	if path, ok := os.LookupEnv(constants.OverwriteDefaultSystemPathEnvVarName); ok {
		return path, nil
	}
	return defaultSystemInstallPath()
}
