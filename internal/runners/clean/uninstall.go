package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/svcctl"
)

type confirmAble interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
}

type Uninstall struct {
	out     output.Outputer
	confirm confirmAble
	cfg     configurable
	ipComm  svcctl.IPCommunicator
}

type UninstallParams struct {
	Force bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Configurer
	primer.IPCommunicator
}

func NewUninstall(prime primeable) (*Uninstall, error) {
	return newUninstall(prime.Output(), prime.Prompt(), prime.Config(), prime.IPComm())
}

func newUninstall(out output.Outputer, confirm confirmAble, cfg configurable, ipComm svcctl.IPCommunicator) (*Uninstall, error) {
	return &Uninstall{
		out:     out,
		confirm: confirm,
		cfg:     cfg,
		ipComm:  ipComm,
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

	if err := stopServices(u.cfg, u.out, u.ipComm, params.Force); err != nil {
		return errs.Wrap(err, "Failed to stop services.")
	}

	return u.runUninstall()
}
