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

func execDir(baseDir ...string) string {
	if len(baseDir) > 0 {
		return baseDir[0]
	}
	path, err := os.Executable()
	if err != nil {
		logging.Error("Could not determine executable directory: %v", err)
		path, err = filepath.Abs(os.Args[0])
		if err != nil {
			logging.Error("Could not get absolute directory of os.Args[0]", err)
		}
	}

	pathEvaled, err := filepath.EvalSymlinks(path)
	if err != nil {
		logging.Error("Could not eval symlinks: %v", err)
	} else {
		path = pathEvaled
	}

	return filepath.Dir(path)
}

func newAppInfo(name, executableBase string, baseDir ...string) *AppInfo {
	return &AppInfo{
		name,
		filepath.Join(execDir(baseDir...), executableBase+osutils.ExeExt),
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
