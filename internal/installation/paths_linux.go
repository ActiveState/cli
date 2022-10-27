package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func InstallPathForBranch(branch string) (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(home, ".local", "ActiveState", "StateTool", branch), nil
}

func defaultSystemInstallPath() (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, ".local", "share", "applications"), nil
}
