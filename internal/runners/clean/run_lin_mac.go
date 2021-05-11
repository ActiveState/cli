// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/logging"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	err = removeInstallDir(u.installPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_install_dir", "Coul dnot remove installation directory")
	}

	err = removeConfig(u.cfg.ConfigPath())
	if err != nil {
		return locale.WrapError(err, "err_clean_config_dir", "Could not remove config directory")
	}

	u.out.Print(locale.T("clean_success_message"))
	return nil
}

func removeConfig(cfg configurable) error {
	file, err := os.Open(logging.FilePath())
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	return os.RemoveAll(cfg.ConfigPath())
}

func removeInstall(installPath string) error {
	return os.Remove(installPath)
}
