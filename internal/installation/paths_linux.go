package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func installPathForBranch(branch string) (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	installPath := filepath.Join(home, ".ActiveState", "StateTool", branch)

	if !isValidInstallPath(installPath) {
		return "", errs.New("Invalid install path: %s", installPath)
	}

	return installPath, nil
}

func defaultSystemInstallPath() (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, ".local", "share", "applications"), nil
}
