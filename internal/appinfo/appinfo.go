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

func TrayApp() *AppInfo {
	return &AppInfo{
		constants.TrayAppName,
		filepath.Join(filepath.Dir(os.Args[0]), "state-tray") + osutils.ExeExt,
	}
}

func StateApp() *AppInfo {
	return &AppInfo{
		constants.StateAppName,
		filepath.Join(filepath.Dir(os.Args[0]), "state") + osutils.ExeExt,
	}
}

func (a *AppInfo) Name() string {
	return a.name
}

func (a *AppInfo) Exec() string {
	return a.executable
}
