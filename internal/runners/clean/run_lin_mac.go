// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

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

func removeStateToolInstall() error {
	installPath, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_uninstall_exec_path", "Could not get executable path")
	}

	return os.Remove(installPath)
}
