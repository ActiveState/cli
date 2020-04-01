// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

func run(params *UninstallParams, confirm confirmAble, outputer output.Outputer) error {
	err := removeCache(params.CachePath)
	if err != nil {
		return err
	}

	err = removeInstall(params.InstallPath)
	if err != nil {
		return err
	}

	err = removeConfig(params.ConfigPath)
	if err != nil {
		return err
	}

	outputer.Print(locale.T("clean_success_message"))
	return nil
}

func removeCache(cachePath string) error {
	logging.Debug("Removing cache path: %s", cachePath)
	return os.RemoveAll(cachePath)
}

func removeConfig(configPath string) error {
	logging.Debug("Removing config directory: %s", configPath)
	if file, ok := logging.CurrentHandler().Output().(*os.File); ok {
		err := file.Sync()
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	}

	return os.RemoveAll(configPath)
}

func removeInstall(installPath string) error {
	logging.Debug("Removing State Tool binary: %s", installPath)
	return os.Remove(installPath)
}
