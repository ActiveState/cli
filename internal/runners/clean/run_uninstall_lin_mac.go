// +build !windows

package clean

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

func runUninstall(params *UninstallParams, confirm confirmAble, outputer output.Outputer) error {
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

func removeConfig(configPath string) error {
	file, err := os.Open(filepath.Join(configPath, "log.txt"))
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
