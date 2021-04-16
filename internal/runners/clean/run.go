package clean

import (
	"errors"
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/shirou/gopsutil/process"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return locale.WrapError(err, "err_clean_cache", "Could not remove cache")
	}

	err = stopTrayApp(u.cfg.GetInt(constants.TrayConfigPid))
	if err != nil {
		return locale.WrapError(err, "err_clean_stop_tray", "Could not stop the state tray application")
	}

	err = stopService()
	if err != nil {
		return locale.WrapError(err, "err_clean_stop_service", "Could not stop state service")
	}

	err = removeTrayApp()
	if err != nil {
		return locale.WrapError(err, "err_clean_tray_app", "could not remove state tray application")
	}

	err = removeAutoStartFile(u.cfg.GetString(constants.AutoStartPath))
	if err != nil {
		return locale.WrapError(err, "err_clean_remove_autostart", "Could not remove autostart file")
	}

	if runtime.GOOS == "windows" {
		err = removeDirs(u.cfg.GetString(constants.InstallPath), u.cfg.ConfigPath())
		if err != nil {
			return locale.WrapError(err, "err_clean_install_dirs", "Could nto remove installation directories")
		}
	} else {
		err = removeInstallDir(u.cfg.GetString(constants.InstallPath))
		if err != nil {
			return locale.WrapError(err, "err_clean_install_dir", "Coul dnot remove installation directory")
		}

		err = removeConfig(u.cfg.ConfigPath())
		if err != nil {
			return locale.WrapError(err, "err_clean_config_dir", "Could not remove config directory")
		}
	}

	u.out.Print(locale.T("clean_success_message"))
	return nil
}

func stopTrayApp(pid int) error {
	if pid <= 0 {
		return nil
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}
		return locale.WrapError(err, "err_clean_pid", "Could not detect if state-tray PID exists")
	}
	if err := proc.Kill(); err != nil {
		return locale.WrapError(err, "err_kill_tray_proc", "Could not kill state-tray")
	}

	return nil
}

func stopService() error {
	svcInfo := appinfo.SvcApp()
	if !fileutils.FileExists(svcInfo.Exec()) {
		return nil
	}

	exitCode, _, err := exeutils.Execute(svcInfo.Exec(), []string{"stop"}, nil)
	if err != nil {
		return locale.WrapError(err, "err_clean_stop_service", "Stopping {{.V0}} return error", svcInfo.Name())
	}
	if exitCode != 0 {
		return locale.WrapError(err, "err_clean_stop_svc_exit_code", "Stopping {{.V0}} exited with code {{.V1}}", svcInfo.Name(), string(exitCode))
	}

	return nil
}

func removeCache(cachePath string) error {
	err := os.RemoveAll(cachePath)
	if err != nil {
		return locale.WrapError(err, "err_remove_cache", "Could not remove State Tool cache directory")
	}
	return nil
}

func removeAutoStartFile(path string) error {
	if path == "" || !fileutils.FileExists(path) {
		return nil
	}

	err := os.Remove(path)
	if err != nil {
		return locale.WrapError(err, "err_remove_auto_start", "Could not remove auto start file at path: {{.V0}}", path)
	}
	return nil
}
