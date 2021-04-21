package installation

import (
	"os/user"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
)

func defaultInstallPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", errs.Wrap(err, "Could not access info on current user")
	}
	return filepath.Join(usr.HomeDir, ".local", "ActiveState", "StateTool", constants.BranchName), nil
}
