//go:build !windows
// +build !windows

package clean

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

func (u *Uninstall) runUninstall() error {
	// we aggregate installation errors, such that we can display all installation problems in the end
	// TODO: This behavior should be replaced with a proper rollback mechanism https://www.pivotaltracker.com/story/show/178134918
	var aggErr error
	err := removeCache(storage.CachePath())
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_cache_err", "Failed to remove cache directory {{.V0}}.", storage.CachePath())
	}

	err = undoPrepare(u.cfg)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_prepare_err", "Failed to undo some installation steps.")
	}

	err = removeInstall(u.cfg)
	fmt.Println("Remove err:", errs.JoinMessage(err))
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory {{.V0}}", filepath.Dir(appinfo.StateApp().Exec()))
	}

	err = removeEnvPaths(u.cfg)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_paths_err", "Failed to remove PATH entries from environment")
	}

	if aggErr != nil {
		return aggErr
	}

	path := u.cfg.ConfigPath()
	if err := u.cfg.Close(); err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_close_config", "Could not stop config database connection.")
	}

	err = removeConfig(path, u.out)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_config_err", "Failed to remove configuration directory {{.V0}}", u.cfg.ConfigPath())

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

func removeInstall(cfg configurable) error {
	stateInfo := appinfo.StateApp()
	stateSvcInfo := appinfo.SvcApp()
	stateTrayInfo := appinfo.TrayApp()

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := installation.StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", stateTrayInfo.Name())
	}

	var aggErr error

	for _, info := range []*appinfo.AppInfo{stateInfo, stateSvcInfo, stateTrayInfo} {
		if err := os.Remove(info.LegacyExec()); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				aggErr = errs.Wrap(aggErr, "Could not remove (legacy) %s: %v", info.LegacyExec(), err)
			}
		}

		err := os.Remove(info.Exec())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			aggErr = errs.Wrap(aggErr, "Could not remove %s: %v", info.Exec(), err)
		}
	}

	if transitionalStatePath := cfg.GetString(installation.CfgTransitionalStateToolPath); transitionalStatePath != "" {
		if err := os.Remove(transitionalStatePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			aggErr = errs.Wrap(aggErr, "Could not remove %s: %v", transitionalStatePath, err)
		}
	}

	appPath, err := installation.LauncherInstallPath()
	if err != nil {
		return errs.Wrap(aggErr, "Could not determine OS specific launcher install path")
	}

	if err := installation.RemoveSystemFiles(appPath); err != nil {
		aggErr = errs.Wrap(aggErr, "Failed to remove system files at %s: %v", appPath, err)
	}

	installPath, err := installation.InstallPath()
	if err != nil {
		aggErr = errs.Wrap(aggErr, "Could not get installation path")
	}

	if fileutils.DirExists(installPath) {
		empty, err := fileutils.IsEmptyDir(installPath)
		if err == nil && empty {
			removeErr := os.RemoveAll(installPath)
			if err != nil {
				aggErr = errs.Wrap(removeErr, "Could not remove install path")
			}
		} else {
			aggErr = errs.Wrap(aggErr, "Could not check if installation path is empty")
		}
	}

	return aggErr
}

func checkAdmin() error {
	return nil
}

func getAdminInstall() (bool, error) {
	return false, nil
}
