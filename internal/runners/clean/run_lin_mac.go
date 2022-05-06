//go:build !windows
// +build !windows

package clean

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/installmgr"
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
	if errors.Is(err, errDirNotEmpty) {
		u.out.Notice(locale.T("uninstall_warn_not_empty", errs.JoinMessage(err)))
	} else if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory")
	}

	err = removeEnvPaths(u.cfg)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_paths_err", "Failed to remove PATH entries from environment")
	}

	path := u.cfg.ConfigPath()
	if err := u.cfg.Close(); err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_close_config", "Could not stop config database connection.")
	}

	err = removeConfig(path, u.out)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_config_err", "Failed to remove configuration directory {{.V0}}", u.cfg.ConfigPath())

	}

	if aggErr != nil {
		return aggErr
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
	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := installmgr.StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", constants.TrayAppName)
	}

	var aggErr error

	// Get the install path before we remove the actual executable
	// to avoid any errors from this function
	installPath, err := installation.InstallPathFromExecPath()
	if err != nil {
		aggErr = errs.Wrap(aggErr, "Could not get installation path")
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

	if err := installmgr.RemoveSystemFiles(appPath); err != nil {
		aggErr = errs.Wrap(aggErr, "Failed to remove system files at %s: %v", appPath, err)
	}

	if fileutils.DirExists(installPath) {
		err = cleanInstallDir(installPath)
		if err != nil {
			aggErr = errs.Wrap(err, "Could not clean install path")
		}
		if err := removeEmptyDir(installPath); err != nil {
			aggErr = errs.Wrap(err, "Could not remove install path: %s", installPath)
		}
	}

	return aggErr
}

func verifyInstallation() error {
	return nil
}

var errDirNotEmpty = errs.New("Not empty")

func removeEmptyDir(dir string) error {
	empty, err := fileutils.IsEmptyDir(dir)
	if err == nil && empty {
		removeErr := os.RemoveAll(dir)
		if err != nil {
			return errs.Wrap(removeErr, "Could not remove directory")
		}
	} else if err != nil {
		return errs.Wrap(err, "Could not check if directory is empty")
	}

	if !empty {
		return errDirNotEmpty
	}

	return nil
}

func cleanInstallDir(dir string) error {
	stateExec, err := installation.NewExec(installation.StateApp)
	if err != nil {
		return locale.WrapError(err, "err_state_info")
	}

	serviceExec, err := installation.NewExec(installation.ServiceApp)
	if err != nil {
		return locale.WrapError(err, "err_service_info")
	}

	trayExec, err := installation.NewExec(installation.TrayApp)
	if err != nil {
		return locale.WrapError(err, "err_tray_info")
	}

	var asFiles = []string{
		installation.InstallDirMarker,
		constants.StateInstallerCmd + exeutils.Extension,

		// Remove all of the state tool executables and finally the
		// bin directory
		filepath.Join(installation.BinDirName, stateExec),
		filepath.Join(installation.BinDirName, serviceExec),
		filepath.Join(installation.BinDirName, trayExec),
		installation.BinDirName,

		// The system directory is on MacOS only and contains the tray
		// application files. It is safe for us to remove this directory
		// without first inspecting the contents.
		"system",
	}

	for _, file := range asFiles {
		f := filepath.Join(dir, file)

		var err error
		if fileutils.DirExists(f) && fileutils.IsDir(f) {
			err = os.RemoveAll(f)
		} else if fileutils.FileExists(f) {
			err = os.Remove(f)
		}
		if err != nil {
			return errs.Wrap(err, "Could not clean install directory")
		}
	}

	return nil
}
