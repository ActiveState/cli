// +build !windows

package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

func run(params *RunParams, confirm confirmAble, outputer output.Outputer) error {
	logging.Debug("Removing cache path: %s", params.CachePath)
	err := os.RemoveAll(params.CachePath)
	if err != nil {
		return err
	}

	logging.Debug("Removing state tool binary: %s", params.InstallPath)
	err = os.Remove(params.InstallPath)
	if err != nil {
		return err
	}

	logging.Debug("Removing config directory: %s", params.ConfigPath)
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

	err = os.RemoveAll(params.ConfigPath)
	if err != nil {
		return err
	}

	outputer.Print(locale.T("clean_success_message"))
	return nil
}
