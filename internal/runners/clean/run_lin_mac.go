//go:build !windows
// +build !windows

package clean

import (
	"errors"
	"os"
	"path/filepath"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/legacytray"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/strutils"
)

func (u *Uninstall) runUninstall(params *UninstallParams) error {
	// we aggregate installation errors, such that we can display all installation problems in the end
	// TODO: This behavior should be replaced with a proper rollback mechanism https://www.pivotaltracker.com/story/show/178134918
	var aggErr, err error
	if params.All {
		err := removeCache(storage.CachePath())
		if err != nil {
			logging.Debug("Could not remove cache at %s: %s", storage.CachePath(), errs.JoinMessage(err))
			aggErr = locale.WrapError(aggErr, "uninstall_remove_cache_err", "Failed to remove cache directory {{.V0}}.", storage.CachePath())
		}
	}

	err = undoPrepare()
	if err != nil {
		logging.Debug("Could not undo prepare: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "uninstall_prepare_err", "Failed to undo some installation steps.")
	}

	if err := removeApp(); err != nil {
		logging.Debug("Could not remove app: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "uninstall_remove_app_err", "Failed to remove service application")
	}

	err = removeInstall(u.cfg)
	if err != nil {
		if dirNotEmpty := (&dirNotEmptyError{}); errors.As(err, &dirNotEmpty) {
			logging.Debug("Could not remove install as dir is not empty: %s", errs.JoinMessage(err))
			aggErr = locale.WrapError(aggErr, "uninstall_warn_not_empty_already_localized", dirNotEmpty.Error())
		} else {
			logging.Debug("Could not remove install: %s", errs.JoinMessage(err))
			aggErr = locale.WrapError(aggErr, "uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory")
		}
	}

	err = removeEnvPaths(u.cfg)
	if err != nil {
		logging.Debug("Could not remove env paths: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "uninstall_remove_paths_err", "Failed to remove PATH entries from environment")
	}

	if params.All {
		path := u.cfg.ConfigPath()
		if err := u.cfg.Close(); err != nil {
			logging.Debug("Could not close config: %s", errs.JoinMessage(err))
			aggErr = locale.WrapError(aggErr, "uninstall_close_config", "Could not stop config database connection.")
		}

		err = removeConfig(path, u.out)
		if err != nil {
			logging.Debug("Could not remove config: %s", errs.JoinMessage(err))
			aggErr = locale.WrapError(aggErr, "uninstall_remove_config_err", "Failed to remove configuration directory {{.V0}}", u.cfg.ConfigPath())
		}
	}

	if aggErr != nil {
		return aggErr
	}

	u.out.Notice(locale.T("clean_success_message"))
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

	out.Notice(locale.Tl("clean_config_succes", "Successfully removed State Tool config directory"))
	return nil
}

func removeInstall(cfg *config.Instance) error {
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

	if fileutils.DirExists(installPath) {
		err = cleanInstallDir(installPath, cfg)
		if err != nil {
			aggErr = errs.Wrap(err, "Could not clean install path")
		}
		if err := removeEmptyDir(installPath); err != nil {
			aggErr = errs.Wrap(err, "Could not remove install path: %s", installPath)
		}
	}

	return aggErr
}

func removeApp() error {
	svcApp, err := svcApp.New()
	if err != nil {
		return locale.WrapError(err, "err_autostart_app")
	}

	err = svcApp.Uninstall()
	if err != nil {
		return locale.WrapError(err, "err_uninstall_app", "Could not uninstall the State Tool service app.")
	}

	return nil
}

func verifyInstallation() error {
	return nil
}

type dirNotEmptyError struct {
	*locale.LocalizedError
}

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
		content, err := strutils.ParseTemplate(
			"{{- range $file := .Files}}\n - {{$file}}\n{{- end}}",
			map[string]interface{}{"Files": fileutils.ListDirSimple(dir, true)},
			nil)
		if err != nil {
			return errs.Wrap(err, "Could not parse file list template")
		}
		return &dirNotEmptyError{locale.NewInputError("uninstall_warn_not_empty", "", content)}
	}

	return nil
}

func cleanInstallDir(dir string, cfg *config.Instance) error {
	err := legacytray.DetectAndRemove(dir, cfg)
	if err != nil {
		return errs.Wrap(err, "Could not remove legacy tray")
	}

	execs, err := installation.Executables()
	if err != nil {
		return errs.Wrap(err, "Could not get executable paths")
	}

	var asFiles = []string{
		installation.InstallDirMarker,
		constants.StateInstallerCmd + osutils.ExeExtension,
	}

	// Remove all of the state tool executables and finally the
	// bin directory
	asFiles = append(asFiles, execs...)
	asFiles = append(asFiles, installation.BinDirName)

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
