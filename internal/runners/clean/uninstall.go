package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

type confirmAble interface {
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
}

type Uninstall struct {
	out         output.Outputer
	confirm     confirmAble
	ConfigPath  string
	CachePath   string
	InstallPath string
}

type UninstallParams struct {
	Force bool
}

func NewUninstall(outputer output.Outputer, confirmer confirmAble) *Uninstall {
	return &Uninstall{
		out:     outputer,
		confirm: confirmer,
	}
}

func (u *Uninstall) Run(params *UninstallParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_uninstall_activated"))
	}

	if !params.Force {
		ok, fail := u.confirm.Confirm(locale.T("uninstall_confirm"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	return u.runUninstall()
}
