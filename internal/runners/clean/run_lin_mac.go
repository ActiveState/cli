// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func removeDirs(dirs ...string) error {
	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		if err != nil {
			return locale.WrapError(err, "err_uninstall_remove_dir", "Could not remove directory")
		}
	}
}

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

func removeInstallDir(dir string) error {
	if dir == "" {
		return locale.NewError("err_uninstall_no_dir", "Could not remove installing directory, not set in config")
	}

	err := os.RemoveAll(dir)
	if err != nil {
		return locale.WrapError(err, "err_remove_install_dir", "Could not remove installation directory")
	}
	return nil
}
