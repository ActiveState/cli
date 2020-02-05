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

	return run(params, c.confirm, c.out)
}
