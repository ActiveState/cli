// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func removeConfig(configPath string) error {
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

	return os.RemoveAll(configPath)
}

func (u *Uninstall) removeInstallDir() error {
	path := u.cfg.GetString(constants.InstallPath)
	if path == "" {
		return locale.NewError("err_uninstall_no_dir", "Could not remove installing directory, not set in config")
	}

	err := os.RemoveAll(u.cfg.GetString(constants.InstallPath))
	if err != nil {
		return locale.WrapError(err, "err_remove_install_dir", "Could not remove installation directory")
	}
	return nil
}
