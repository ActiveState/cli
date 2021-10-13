package installation

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
)

func DefaultInstallPath() (string, error) {
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "ActiveState", "StateTool", constants.BranchName), nil
}

func defaultSystemInstallPath() (string, error) {
	// There is no system install path for Windows
	return "", nil
}
