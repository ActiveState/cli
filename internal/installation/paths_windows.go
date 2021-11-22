package installation

import (
	"os"
	"path/filepath"
)

func InstallPathForBranch(branch string) (string, error) {
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "ActiveState", "StateTool", branch), nil
}

func defaultSystemInstallPath() (string, error) {
	// There is no system install path for Windows
	return "", nil
}
