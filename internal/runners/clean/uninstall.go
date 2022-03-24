package clean

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type confirmAble interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
}

type Uninstall struct {
	out     output.Outputer
	confirm confirmAble
	cfg     configurable
	an      analytics.Dispatcher
}

type UninstallParams struct {
	Force bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Configurer
	primer.Analyticer
}

func NewUninstall(prime primeable) (*Uninstall, error) {
	return newUninstall(prime.Output(), prime.Prompt(), prime.Config(), prime.Analytics())
}

func newUninstall(out output.Outputer, confirm confirmAble, cfg configurable, an analytics.Dispatcher) (*Uninstall, error) {
	return &Uninstall{
		out:     out,
		confirm: confirm,
		cfg:     cfg,
		an:      an,
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

	err := verifyInstallation()
	if err != nil {
		return errs.Wrap(err, "Could not verify installation")
	}

	if err := stopServices(u.cfg, u.out, params.Force); err != nil {
		return errs.Wrap(err, "Failed to stop services.")
	}

	err = u.runUninstall()
	if err != nil {
		return errs.Wrap(err, "Could not complete uninstallation")
	}
	events.WaitForEvents(5*time.Second, u.an.Close, logging.Close)

	return nil
}
