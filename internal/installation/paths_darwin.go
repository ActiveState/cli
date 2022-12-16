package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/osutils/user"
)

func installPathForBranch(branch string) (string, error) {
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

	return filepath.Join(home, "Applications"), nil
}
