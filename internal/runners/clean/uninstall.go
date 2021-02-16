package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type confirmAble interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
}

type Uninstall struct {
	out         output.Outputer
	confirm     confirmAble
	cfg         configurable
	installPath string
}

type UninstallParams struct {
	Force bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Configurer
}

func NewUninstall(prime primeable) (*Uninstall, error) {
	return newUninstall(prime.Output(), prime.Prompt(), prime.Config())
}

func newUninstall(out output.Outputer, confirm confirmAble, cfg configurable) (*Uninstall, error) {
	installPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return &Uninstall{
		out:         out,
		confirm:     confirm,
		installPath: installPath,
		cfg:         cfg,
	}, nil
}

func (u *Uninstall) Run(params *UninstallParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return locale.NewError("err_uninstall_activated")
	}

	if !params.Force {
		ok, err := u.confirm.Confirm(locale.T("confirm"), locale.T("uninstall_confirm"), new(bool))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	return u.runUninstall()
}
