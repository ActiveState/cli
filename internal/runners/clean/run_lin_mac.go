// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/logging"
)

func removeConfig(configPath string) error {
	config.Get().SkipSave(true)

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

func removeInstall(installPath string) error {
	return os.Remove(installPath)
}
