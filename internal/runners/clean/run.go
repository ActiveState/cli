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

	err = u.removeInstall()
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

func (u *Uninstall) removeAutoStartFile() error {
	autoStartPath := u.cfg.GetString(constants.AutoStartPath)
	if !fileutils.FileExists(autoStartPath) {
		return nil
	}

	err := os.Remove(autoStartPath)
	if err != nil {
		return locale.WrapError(err, "err_remove_auto_start", "Could not remove auto start file at path: {{.V0}}", autoStartPath)
	}
	return nil
}
