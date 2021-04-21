package installation

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
)

func defaultInstallPath() (string, error) {
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "ActiveState", "StateTool", constants.BranchName), nil
}
