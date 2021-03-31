package installer

import (
	"os"
	"path/filepath"
)

func defaultInstallPath() string {
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "ActiveState", "StateTool")
}
