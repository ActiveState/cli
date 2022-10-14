package offinstall

import (
	"os"
	"path/filepath"
)

func DefaultInstallPath() (string, error) {
	// There is no system install path for Windows
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Programs"), nil
}
