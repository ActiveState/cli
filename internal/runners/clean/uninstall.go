package clean

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type confirmAble interface {
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
}

type Uninstall struct {
	out         output.Outputer
	confirm     confirmAble
	configPath  string
	cachePath   string
	installPath string
}

type UninstallParams struct {
	Force bool
}

type primeable interface {
	primer.Outputer
	Prompt() confirmAble
}

func NewUninstall(prime primeable) (*Uninstall, error) {
	installPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return &Uninstall{
		out:         prime.Output(),
		confirm:     prime.Prompt(),
		installPath: installPath,
		configPath:  config.ConfigPath(),
		cachePath:   config.CachePath(),
	}, nil
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
