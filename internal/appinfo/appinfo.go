package appinfo

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

type AppInfo struct {
	name       string
	executable string
}

func TrayApp() *AppInfo {
	return &AppInfo{
		constants.TrayAppName,
		exePath("state-tray"),
	}
}

func StateApp() *AppInfo {
	return &AppInfo{
		constants.StateAppName,
		exePath("state"),
	}
}

func (a *AppInfo) Name() string {
	return a.name
}

func (a *AppInfo) Exec() string {
	return a.executable
}

func exePath(exeName string) string {
	fallback := filepath.Join(filepath.Dir(os.Args[0]), exeName+osutils.ExeExt)

	path, err := os.Executable()
	if err != nil {
		logging.Errorf("Could not get executable path for: %v", err)
		return fallback
	}

	pathEvaled, err := filepath.EvalSymlinks(path)
	if err != nil {
		logging.Error("Could not eval symlinks: %v", err)
	} else {
		path = pathEvaled
	}

	return filepath.Join(filepath.Dir(path), exeName+osutils.ExeExt)
}