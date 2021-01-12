package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	err = removeInstall(u.installPath)
	if err != nil {
		return err
	}

	err = removeConfig(u.cfg)
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
