package clean

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

type confirmAble interface {
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
}

type Clean struct {
	out     output.Outputer
	confirm confirmAble
}

type RunParams struct {
	Force      bool
	ConfigPath string
	CachePath  string
}

func NewClean(outputer output.Outputer, confirmer confirmAble) *Clean {
	return &Clean{
		out:     outputer,
		confirm: confirmer,
	}
}

func (c *Clean) Run(params *RunParams) error {
	return c.run(params)

}

func (c *Clean) run(params *RunParams) error {
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

	installPath, err := getInstallPath()
	if err != nil {
		return err
	}

	logging.Debug("Removing config directory: %s", params.ConfigPath)
	logging.Debug("Removing cache path: %s", params.CachePath)
	logging.Debug("Removing state tool binary: %s", installPath)
	if file, ok := logging.CurrentHandler().Output().(*os.File); ok {
		err = file.Sync()
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

	err = os.RemoveAll(params.CachePath)
	if err != nil {
		return err
	}

	err = os.Remove(installPath)
	if err != nil {
		return err
	}

	c.out.Print(locale.T("clean_success_message"))
	return nil
}

func getInstallPath() (string, error) {
	var finder string
	switch runtime.GOOS {
	case "linux", "darwin":
		finder = "which"
	case "windows":
		finder = "where"
	default:
		return "", errors.New(locale.Tr("err_clean_unsupported_platform", runtime.GOOS))
	}

	cmd := exec.Command(finder, constants.CommandName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
