// +build !windows

package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func removeConfig(cfg configurable) error {
	file, err := os.Open(logging.FilePath())
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	return os.RemoveAll(cfg.ConfigPath())
}

func removeInstall(installDir string) error {
	stateInfo := appinfo.StateApp(installDir)
	stateSvcInfo := appinfo.SvcApp(installDir)
	stateTrayInfo := appinfo.TrayApp(installDir)

	for _, info := range []*appinfo.AppInfo{stateInfo, stateSvcInfo, stateTrayInfo} {
		err := os.Remove(info.Exec())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return locale.WrapError(err, "err_remove_install", "Failed to remove executable {{.V0}}", info.Exec())
		}
	}

	appPath, err := installation.LauncherInstallPath()
	if err != nil {
		return errs.Wrap(err, "Could not determine OS specific launcher install path")
	}

	return installation.RemoveSystemFiles(appPath)
}
