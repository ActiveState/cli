package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/shirou/gopsutil/process"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return errs.Wrap(err, "Could not remove cache")
	}

	err = stopTrayApp(u.cfg.GetInt(constants.TrayConfigPid))
	if err != nil {
		return errs.Wrap(err, "Could not stop a state tray application")
	}

	err = stopService()
	if err != nil {
		return errs.Wrap(err, "Could not stop state service")
	}

	err = removeTrayApp()
	if err != nil {
		return errs.Wrap(err, "Could not remove tray application")
	}

	err = removeAutoStartFile(u.cfg.GetString(constants.AutoStartPath))
	if err != nil {
		return errs.Wrap(err, "Could not remove auto start file")
	}

	err = removeInstallDir(u.cfg.GetString(constants.InstallPath))
	if err != nil {
		return errs.Wrap(err, "Could not remove installation directory")
	}

	err = removeConfig(u.cfg.ConfigPath())
	if err != nil {
		return errs.Wrap(err, "Could not remove config directory")
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
		return errs.Wrap(err, "Could not detect if state-tray pid exists")
	}
	if err := proc.Kill(); err != nil {
		return errs.Wrap(err, "Could not kill state-tray")
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
		return errs.Wrap(err, "Stopping %s returned error", svcInfo.Name())
	}
	if exitCode != 0 {
		return errs.New("Stopping %s exited with code %d", svcInfo.Name(), exitCode)
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
