//go:build windows
// +build windows

package clean

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/legacytray"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func (u *Uninstall) runUninstall(params *UninstallParams) error {
	// we aggregate installation errors, such that we can display all installation problems in the end
	// TODO: This behavior should be replaced with a proper rollback mechanism https://www.pivotaltracker.com/story/show/178134918
	var aggErr error
	logFile, err := ioutil.TempFile("", "state-clean-uninstall")
	if err != nil {
		logging.Error("Could not create temporary log file: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "err_clean_logfile", "Could not create temporary log file")
	}

	stateExec, err := installation.StateExec()
	if err != nil {
		logging.Debug("Could not get State Tool executable: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "err_state_exec")
	}

	err = removeInstall(logFile.Name(), params, u.cfg)
	if err != nil {
		logging.Debug("Could not remove installation: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory {{.V0}}", filepath.Dir(stateExec))
	}

	err = removeApp()
	if err != nil {
		logging.Debug("Could not remove app: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "uninstall_remove_app_err", "Failed to remove service application")
	}

	if params.All {
		err = removeCache(storage.CachePath())
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

	err = removeEnvPaths(u.cfg)
	if err != nil {
		logging.Debug("Could not remove environment paths: %s", errs.JoinMessage(err))
		aggErr = locale.WrapError(aggErr, "uninstall_remove_paths_err", "Failed to remove PATH entries from environment")
	}

	if aggErr != nil {
		return aggErr
	}

	u.out.Notice(locale.Tr("clean_message_windows", logFile.Name()))
	if params.Prompt {
		u.out.Print(locale.Tl("clean_uninstall_confirm_exit", "Press enter to exit."))
		fmt.Scanln(ptr.To("")) // Wait for input from user
	}
	return nil
}

func removeConfig(configPath string, out output.Outputer) error {
	logFile, err := ioutil.TempFile("", "state-clean-config")
	if err != nil {
		return locale.WrapError(err, "err_clean_logfile", "Could not create temporary log file")
	}

	out.Notice(locale.Tr("clean_config_message_windows", logFile.Name()))
	return removePaths(logFile.Name(), configPath)
}

func removeInstall(logFile string, params *UninstallParams, cfg *config.Instance) error {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return locale.WrapError(err, "err_service_exec")
	}

	err = legacytray.DetectAndRemove(filepath.Dir(svcExec), cfg)
	if err != nil {
		return locale.WrapError(err, "err_remove_legacy_tray", "Could not remove legacy tray application")
	}

	err = os.Remove(svcExec)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return locale.WrapError(err, "uninstall_rm_exec", "Could not remove executable: {{.V0}}. Error: {{.V1}}.", svcExec, err.Error())
	}

	stateExec, err := installation.StateExec()
	if err != nil {
		return locale.WrapError(err, "err_state_exec")
	}

	// Schedule removal of the entire branch name directory.
	// This is because Windows often thinks the installation.InstallDirMarker and
	// constants.StateInstallerCmd files are still in use.
	branchDir := filepath.Dir(filepath.Dir(stateExec))
	paths := []string{stateExec, branchDir}
	if params.All {
		paths = append(paths, cfg.ConfigPath()) // also remove the config directory
	}
	// If the transitional state tool path is known, we remove it. This is done in the background, because the transitional State Tool can be the initiator of the uninstall request
	if transitionalStateTool := cfg.GetString(installation.CfgTransitionalStateToolPath); transitionalStateTool != "" {
		paths = append(paths, transitionalStateTool)
	}

	return removePaths(logFile, paths...)
}

func removePaths(logFile string, paths ...string) error {
	logging.Debug("Removing paths: %v", paths)
	scriptName := "removePaths"
	scriptBlock, err := assets.ReadFileBytes(fmt.Sprintf("scripts/%s.bat", scriptName))
	if err != nil {
		return err
	}
	sf, err := scriptfile.New(language.Batch, scriptName, string(scriptBlock))
	if err != nil {
		return locale.WrapError(err, "err_clean_script", "Could not create new scriptfile")
	}

	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_clean_executable", "Could not get executable name")
	}

	args := []string{"/C", sf.Filename(), logFile, fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe)}
	args = append(args, paths...)

	_, err = osutils.ExecuteAndForget("cmd.exe", args)
	if err != nil {
		return locale.WrapError(err, "err_clean_start", "Could not start remove direcotry script")
	}

	return nil
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

// verifyInstallation ensures that the State Tool was installed in a way
// that will allow us to properly uninstall
func verifyInstallation() error {
	installationContext, err := installation.GetContext()
	if err != nil {
		return errs.Wrap(err, "Could not check if initial installation was run as admin")
	}

	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not check if current user is an administrator")
	}

	if installationContext.InstalledAsAdmin && !isAdmin {
		return locale.NewInputError("err_uninstall_privilege_mismatch")
	}

	return nil
}
