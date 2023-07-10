package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func installPathForBranch(branch string) (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not determine home directory")
	}
	installPath := filepath.Join(home, "AppData", "Local", "ActiveState", "StateTool", branch)

	if !isValidInstallPath(installPath) {
		return "", errs.New("Invalid install path: %s", installPath)
	}

	return installPath, nil
}

func defaultSystemInstallPath() (string, error) {
	// There is no system install path for Windows
	return "", nil
}
