package clean

import (
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

type confirmAble interface {
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
}

type Clean struct {
	out     output.Outputer
	confirm confirmAble
}

type RunParams struct {
	Force       bool
	ConfigPath  string
	CachePath   string
	InstallPath string
}

func NewClean(outputer output.Outputer, confirmer confirmAble) *Clean {
	return &Clean{
		out:     outputer,
		confirm: confirmer,
	}
}

func (c *Clean) Run(params *RunParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_activated"))
	}

	if !params.Force {
		ok, fail := c.confirm.Confirm(locale.T("clean_confirm_remove"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	if runtime.GOOS == "windows" {
		box := packr.NewBox("../../../assets/scripts/")
		scriptBlock := box.String("clean.bat")
		sf, fail := scriptfile.New(language.Batch, "clean", scriptBlock)
		if fail != nil {
			return fail.ToError()
		}
		cmd := exec.Command("cmd.exe", "/C", sf.Filename(), params.CachePath, params.ConfigPath, params.InstallPath)
		err := cmd.Start()
		if err != nil {
			return err
		}
		return nil
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

	c.out.Print(locale.T("clean_success_message"))
	return nil
}
