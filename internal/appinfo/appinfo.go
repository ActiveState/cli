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

func newAppInfo(name, executableBase string, baseDir ...string) *AppInfo {
	return &AppInfo{
		name,
		exePath(executableBase, baseDir...),
	}
}

func TrayApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.TrayAppName, "state-tray", baseDir...)
}

func StateApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.StateAppName, "state", baseDir...)
}

func SvcApp(baseDir ...string) *AppInfo {
	return newAppInfo(constants.SvcAppName, "state-svc", baseDir...)
}

func (a *AppInfo) Name() string {
	return a.name
}

func (a *AppInfo) Exec() string {
	return a.executable
}

func exePath(exeName string, baseDir ...string) string {
	if len(baseDir) > 0 {
		return filepath.Join(baseDir[0], exeName)
	}

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
