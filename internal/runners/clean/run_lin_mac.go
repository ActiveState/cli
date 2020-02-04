// +build !windows

package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

func run(params *RunParams, confirm confirmAble, outputer output.Outputer) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_activated"))
	}

	if !params.Force {
		ok, fail := confirm.Confirm(locale.T("clean_confirm_remove"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

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
