package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.CachePath)
	if err != nil {
		return err
	}

	err = removeInstall(u.InstallPath)
	if err != nil {
		return err
	}

	err = removeConfig(u.ConfigPath)
	if err != nil {
		return err
	}

	u.out.Print(locale.T("clean_success_message"))
	return nil
}

func removeCache(cachePath string) error {
	return os.RemoveAll(cachePath)
}
