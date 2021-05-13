// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	err = removeInstall(u.installPath)
	if err != nil {
		return locale.WrapError(err, "err_clean_install_dir", "Coul dnot remove installation directory")
	}

	err = removeConfig(u.cfg.ConfigPath(), u.out)
	if err != nil {
		return locale.WrapError(err, "err_clean_config_dir", "Could not remove config directory")
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

	out.Print(locale.Tl("clean_config_succes", "Succsessfully removed State Tool config directory"))
	return nil
}

func removeInstall(installPath string) error {
	return os.Remove(installPath)
}
