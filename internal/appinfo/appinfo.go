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
	e, err := os.Executable()
	if err != nil {
		logging.Debug("Could not determine executable directory: %v", err)
		e, _ = filepath.Abs(os.Args[0])
	}

	return filepath.Base(e)
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
