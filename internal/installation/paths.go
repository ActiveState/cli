package installation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
)

const (
	// CfgTransitionalStateToolPath is the configuration key for the path where a transitional State Tool might still be stored
	CfgTransitionalStateToolPath = "transitional_installation_path"

	BinDirName = "bin"

	InstallDirMarker = ".state_install_root"
)

type InstallMarkerMeta struct {
	Channel  string `json:"channel"`
	Version string `json:"version"`
}

type StateExeDoesNotExistError struct{ *errs.WrapperError }

func IsStateExeDoesNotExistError(err error) bool {
	return errs.Matches(err, &StateExeDoesNotExistError{})
}

func DefaultInstallPath() (string, error) {
	return InstallPathForChannel(constants.ChannelName)
}

// InstallPathForBranch gets the installation path for the given channel.
func InstallPathForChannel(channel string) (string, error) {
	if v := os.Getenv(constants.InstallPathOverrideEnvVarName); v != "" {
		return filepath.Clean(v), nil
	}

	installPath, err := installPathForChannel(channel)
	if err != nil {
		return "", errs.Wrap(err, "Unable to determine install path for channel")
	}

	return installPath, nil
}

func InstallRoot(path string) (string, error) {
	installFile, err := fileutils.FindFileInPath(path, InstallDirMarker)
	if err != nil {
		return "", errs.Wrap(err, "Could not find install marker file in path")
	}

	if !isValidInstallPath(filepath.Dir(installFile)) {
		return "", errs.New("Invalid install path: %s", path)
	}

	return filepath.Dir(installFile), nil
}

func InstallPathFromExecPath() (string, error) {
	exePath := os.Args[0]
	if exe, err := os.Executable(); err == nil {
		exePath = exe
	}

	// Facilitate use-case of running executables from the build dir while developing
	if !condition.BuiltViaCI() && strings.Contains(exePath, string(os.PathSeparator)+"build"+string(os.PathSeparator)) {
		return filepath.Dir(exePath), nil
	}
	if path, ok := os.LookupEnv(constants.OverwriteDefaultInstallationPathEnvVarName); ok {
		return path, nil
	}

	return InstallPathFromReference(filepath.Dir(exePath))
}

func InstallPathFromReference(dir string) (string, error) {
	cmdName := constants.StateCmd + osutils.ExeExtension
	installPath := filepath.Dir(dir)
	binPath, err := BinPathFromInstallPath(installPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not detect installation root")
	}

	stateExe := filepath.Join(binPath, cmdName)
	if !fileutils.TargetExists(stateExe) {
		return "", &StateExeDoesNotExistError{errs.New("Installation bin directory does not contain %s", stateExe)}
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

func ApplicationInstallPath() (string, error) {
	if path, ok := os.LookupEnv(constants.OverwriteDefaultSystemPathEnvVarName); ok {
		return path, nil
	}
	return defaultSystemInstallPath()
}

func isValidInstallPath(path string) bool {
	return fileutils.FileExists(filepath.Join(path, InstallDirMarker))
}
