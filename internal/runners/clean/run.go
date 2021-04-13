package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	// TODO: Stop app before this
	err = removeTrayApp()
	if err != nil {
		return err
	}

	err = removeAutoStartFile(u.cfg.GetString(constants.AutoStartPath))
	if err != nil {
		return err
	}

	err = removeInstallDir(u.cfg.GetString(constants.InstallPath))
	if err != nil {
		return err
	}

	err = removeConfig(u.cfg.ConfigPath())
	if err != nil {
		return err
	}

	u.out.Print(locale.T("clean_success_message"))
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
