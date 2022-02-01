//go:build !windows
// +build !windows

package clean

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
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
	if false {
		err := removeCache(storage.CachePath())
		if err != nil {
			aggErr = locale.WrapError(aggErr, "uninstall_remove_cache_err", "Failed to remove cache directory {{.V0}}.", storage.CachePath())
		}
	}

	err := removeInstall(u.cfg)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory {{.V0}}", filepath.Dir(appinfo.StateApp().Exec()))
	}

	if false {
		err = undoPrepare(u.cfg)
		if err != nil {
			aggErr = locale.WrapError(aggErr, "uninstall_prepare_err", "Failed to undo some installation steps.")
		}

		path := u.cfg.ConfigPath()
		if err := u.cfg.Close(); err != nil {
			aggErr = locale.WrapError(aggErr, "uninstall_close_config", "Could not stop config database connection.")
		}

		err = removeConfig(path, u.out)
		if err != nil {
			aggErr = locale.WrapError(aggErr, "uninstall_remove_config_err", "Failed to remove configuration directory {{.V0}}", u.cfg.ConfigPath())

		}

		err = removeEnvPaths(u.cfg)
		if err != nil {
			aggErr = locale.WrapError(aggErr, "uninstall_remove_paths_err", "Failed to remove PATH entries from environment")
		}

		if aggErr != nil {
			return aggErr
		}
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
	var aggErr error

	for i := 0; i < 2; i++ {
		stateInfo := appinfo.StateApp()
		stateSvcInfo := appinfo.SvcApp()
		stateTrayInfo := appinfo.TrayApp()

		// Todo: https://www.pivotaltracker.com/story/show/177585085
		// Yes this is awkward right now
		if err := installation.StopTrayApp(cfg); err != nil {
			return errs.Wrap(err, "Failed to stop %s", stateTrayInfo.Name())
		}

		for _, info := range []*appinfo.AppInfo{stateInfo, stateSvcInfo, stateTrayInfo} {
			fmt.Println("removing", info.Exec())
			err := os.Remove(info.Exec())
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				aggErr = errs.Wrap(aggErr, "Could not remove %s: %v", info.Exec(), err)
			}
		}

		// this, and the for loop, should be removed after bin dir
		// usage is deprecated
		maybeBinDir := filepath.Dir(stateInfo.Exec())
		if strings.HasSuffix(maybeBinDir, "bin") { // this is dangerous!
			fmt.Println(maybeBinDir)
			if err := os.RemoveAll(maybeBinDir); err != nil {
				aggErr = errs.Wrap(aggErr, "Could not remove directory %s: %v", maybeBinDir, err)
			}
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

	return aggErr
}
