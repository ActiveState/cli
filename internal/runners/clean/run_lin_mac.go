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
	"github.com/ActiveState/cli/internal/output"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	err = removeInstall(u.installPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_install_dir", "Coul dnot remove installation directory")
	}

	err = removeConfig(u.cfg.ConfigPath(), u.out)
	if err != nil {
		return locale.WrapError(err, "err_clean_config_dir", "Could not remove config directory")
	}

	u.out.Print(locale.T("clean_success_message"))
	return nil
}

func removeConfig(configPath string, out output.Outputer) error {
	file, err := os.Open(logging.FilePath())
	if err != nil {
		return locale.WrapError(err, "err_clean_open_log", "Could not open logging file at: {{.V0}}", logging.FilePath())
	}
	err = file.Sync()
	if err != nil {
		return locale.WrapError(err, "err_clean_sync", "Could not sync logging file")
	}
	err = file.Close()
	if err != nil {
		return locale.WrapError(err, "err_clean_close", "Could not close logging file")
	}

	err = os.RemoveAll(configPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_config_remove", "Could not remove config directory")
	}

	out.Print(locale.Tl("clean_config_succes", "Successfully removed State Tool config directory"))
	return nil
}

func removeInstall(_ configurable, logFile, installDir, _ string) error {
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
