package clean

import (
	"os"

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

func removeCache(cachePath string) error {
	return os.RemoveAll(cachePath)
}
