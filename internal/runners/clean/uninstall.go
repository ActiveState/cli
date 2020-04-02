package clean

import (
	"errors"
	"os"
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

type Uninstall struct {
	out     output.Outputer
	confirm confirmAble
}

type UninstallParams struct {
	Force       bool
	ConfigPath  string
	CachePath   string
	InstallPath string
}

func NewUninstall(outputer output.Outputer, confirmer confirmAble) *Uninstall {
	return &Uninstall{
		out:     outputer,
		confirm: confirmer,
	}
}

func (c *Uninstall) Run(params *UninstallParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_uninstall_activated"))
	}

	if !params.Force {
		ok, fail := c.confirm.Confirm(locale.T("uninstall_confirm"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	logging.Debug(
		"Cleaning the following paths:\n %s",
		strings.Join([]string{params.CachePath, params.ConfigPath, params.InstallPath}, "\n "),
	)
	return runUninstall(params, c.confirm, c.out)
}
