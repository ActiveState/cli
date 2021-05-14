package installation

import (
	"os/user"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/mitchellh/go-homedir"
)

func defaultInstallPath() (string, error) {
	// This should actually go to ~/Applications/ActiveStateDesktop.app/Contents/..
	// but we don't have our app yet
	usr, err := user.Current()
	if err != nil {
		return "", errs.Wrap(err, "Could not access info on current user")
	}
	return filepath.Join(usr.HomeDir, ".local", "ActiveState", "StateTool", constants.BranchName), nil
}

func defaultSystemInstallPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, "Applications"), nil
}
