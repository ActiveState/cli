package clean

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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
	cfg     *config.Instance
	ipComm  svcctl.IPCommunicator
	an      analytics.Dispatcher
}

type UninstallParams struct {
	Force          bool
	NonInteractive bool
	All            bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Configurer
	primer.IPCommunicator
	primer.Analyticer
}

func NewUninstall(prime primeable) (*Uninstall, error) {
	return newUninstall(prime.Output(), prime.Prompt(), prime.Config(), prime.IPComm(), prime.Analytics())
}

func newUninstall(out output.Outputer, confirm confirmAble, cfg *config.Instance, ipComm svcctl.IPCommunicator, an analytics.Dispatcher) (*Uninstall, error) {
	return &Uninstall{
		out:     out,
		confirm: confirm,
		cfg:     cfg,
		ipComm:  ipComm,
		an:      an,
	}, nil
}

func (u *Uninstall) Run(params *UninstallParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return locale.NewError("err_uninstall_activated")
	}

	if !params.Force {
		defaultChoice := params.NonInteractive
		confirmMessage := locale.T("uninstall_confirm")
		if params.All {
			confirmMessage = locale.T("uninstall_confirm_all")
		}
		ok, err := u.confirm.Confirm(locale.T("confirm"), confirmMessage, &defaultChoice)
		if err != nil {
			return locale.WrapError(err, "err_uninstall_confirm", "Could not confirm uninstall choice")
		}
		if !ok {
			return locale.NewInputError("err_uninstall_aborted", "Uninstall aborted by user")
		}
	}

	err := verifyInstallation()
	if err != nil {
		return errs.Wrap(err, "Could not verify installation")
	}

	if err := stopServices(u.cfg, u.out, u.ipComm, params.Force); err != nil {
		return errs.Wrap(err, "Failed to stop services.")
	}

	err = u.runUninstall(params)
	if err != nil {
		return errs.Wrap(err, "Could not complete uninstallation")
	}
	events.WaitForEvents(5*time.Second, u.an.Close, logging.Close)

	return nil
}
