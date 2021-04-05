package appinfo

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
)

type AppInfo struct {
	name       string
	executable string
}

func (a *AppInfo) Name() string {
	return a.name
}

func (a *AppInfo) Exec() string {
	return a.executable
}

func TrayApp() (*AppInfo, error) {
	installPath, err := installation.InstallPath()
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect installation path")
	}

	return &AppInfo{
		constants.TrayAppName,
		filepath.Join(installPath, "state-tray") + osutils.ExeExt,
	}, nil
}

func SvcApp() (*AppInfo, error) {
	installPath, err := installation.InstallPath()
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect installation path")
	}

	return &AppInfo{
		constants.SvcAppName,
		filepath.Join(installPath, "state-svc") + osutils.ExeExt,
	}, nil
}

