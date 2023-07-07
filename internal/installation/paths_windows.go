package installation

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
)

func installPathForBranch(branch string) (string, error) {
	installPath := filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "ActiveState", "StateTool", branch)

	if !isValidInstallPath(installPath) {
		return "", errs.New("Invalid install path: %s", installPath)
	}

	return installPath, nil
}

func defaultSystemInstallPath() (string, error) {
	// There is no system install path for Windows
	return "", nil
}
