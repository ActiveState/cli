package appinfo

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils"
)

type AppInfo struct {
	name       string
	executable string
}

func newAppInfo(name, executableBase string, baseDir ...string) *AppInfo {
	dir := filepath.Dir(os.Args[0])
	if len(baseDir) > 0 {
		dir = baseDir[0]
	}
	return &AppInfo{
		name,
		filepath.Join(dir, executableBase+osutils.ExeExt),
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
