// +build !windows

package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
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

	var aggErr error

	for _, info := range []*appinfo.AppInfo{stateInfo, stateSvcInfo, stateTrayInfo} {
		err := os.Remove(info.Exec())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			aggErr = errs.Wrap(aggErr, "Could not remove %s: %v", info.Exec(), err)
		}
	}

	appPath, err := installation.LauncherInstallPath()
	if err != nil {
		return errs.Wrap(aggErr, "Could not determine OS specific launcher install path")
	}

	if err := installation.RemoveSystemFiles(appPath); err != nil {
		aggErr = errs.Wrap(aggErr, "Failed to remove system files at %s: %v", appPath, err)
	}

	return aggErr
}
