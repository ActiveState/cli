package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func installPathForChannel(channel string) (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(home, ".local", "ActiveState", "StateTool", channel), nil
}

func defaultSystemInstallPath() (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, "Applications"), nil
}
