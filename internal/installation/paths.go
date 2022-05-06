package installation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
)

const (
	// CfgTransitionalStateToolPath is the configuration key for the path where a transitional State Tool might still be stored
	CfgTransitionalStateToolPath = "transitional_installation_path"

	BinDirName = "bin"

	InstallDirMarker = ".state_install_root"
)

func DefaultInstallPath() (string, error) {
	return InstallPathForBranch(constants.BranchName)
}

func InstallRoot(path string) (string, error) {
	installFile, err := fileutils.FindFileInPath(path, InstallDirMarker)
	if err != nil {
		return "", errs.Wrap(err, "Could not find install marker file in path")
	}

	return filepath.Dir(installFile), nil
}

func InstallPathFromExecPath() (string, error) {
	exePath := os.Args[0]
	if exe, err := os.Executable(); err == nil {
		exePath = exe
	}

	// Facilitate use-case of running executables from the build dir while developing
	if !condition.BuiltViaCI() && strings.Contains(exePath, "/build/") {
		return filepath.Dir(exePath), nil
	}
	if path, ok := os.LookupEnv(constants.OverwriteDefaultInstallationPathEnvVarName); ok {
		return path, nil
	}

	return InstallPathFromReference(filepath.Dir(exePath))
}

func InstallPathFromReference(dir string) (string, error) {
	cmdName := constants.StateCmd + exeutils.Extension
	installPath := filepath.Dir(dir)
	binPath, err := BinPathFromInstallPath(installPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not detect installation root")
	}

	stateExe := filepath.Join(binPath, cmdName)
	if !fileutils.TargetExists(stateExe) {
		return "", errs.New("Installation bin directory does not contain %s", stateExe)
	}

	return filepath.Dir(binPath), nil
}

func BinPathFromInstallPath(installPath string) (string, error) {
	if installPath == "" {
		return "", errs.New("Cannot detect bin path empty install path")
	}

	var err error
	installPath, err = InstallRoot(installPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not detect install root")
	}

	return filepath.Join(installPath, BinDirName), nil
}

func LauncherInstallPath() (string, error) {
	if path, ok := os.LookupEnv(constants.OverwriteDefaultSystemPathEnvVarName); ok {
		return path, nil
	}
	return defaultSystemInstallPath()
}
