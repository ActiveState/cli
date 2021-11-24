package installation

import (
	"os/user"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/mitchellh/go-homedir"
)

func InstallPathForBranch(branch string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", errs.Wrap(err, "Could not access info on current user")
	}
	return filepath.Join(usr.HomeDir, ".local", "ActiveState", "StateTool", branch), nil
}

func defaultSystemInstallPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, "Applications"), nil
}
