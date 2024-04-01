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

type promptable interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
	Select(title, message string, choices []string, defaultChoice *string) (string, error)
}

type Uninstall struct {
	out    output.Outputer
	prompt promptable
	cfg    *config.Instance
	ipComm svcctl.IPCommunicator
	an     analytics.Dispatcher
}

type UninstallParams struct {
	Force          bool
	NonInteractive bool
	All            bool
	Prompt         bool
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

func newUninstall(out output.Outputer, prompt promptable, cfg *config.Instance, ipComm svcctl.IPCommunicator, an analytics.Dispatcher) (*Uninstall, error) {
	return &Uninstall{
		out:    out,
		prompt: prompt,
		cfg:    cfg,
		ipComm: ipComm,
		an:     an,
	}, nil
}

func (u *Uninstall) Run(params *UninstallParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return locale.NewInputError("err_uninstall_activated")
	}

	err := verifyInstallation()
	if err != nil {
		return errs.Wrap(err, "Could not verify installation")
	}

	if params.Prompt {
		choices := []string{
			locale.Tl("uninstall_prompt", "Uninstall the State Tool, but keep runtime cache and configuration files"),
			locale.Tl("uninstall_prompt_all", "Completely uninstall the State Tool, including runtime cache and configuration files"),
		}
		selection, err := u.prompt.Select("", "", choices, new(string))
		if err != nil {
			return locale.WrapError(err, "err_uninstall_prompt", "Could not read uninstall option")
		}
		if selection == choices[1] {
			params.All = true
		}
	} else if !params.Force {
		defaultChoice := params.NonInteractive
		confirmMessage := locale.T("uninstall_confirm")
		if params.All {
			confirmMessage = locale.T("uninstall_confirm_all")
		}
		ok, err := u.prompt.Confirm(locale.T("confirm"), confirmMessage, &defaultChoice)
		if err != nil {
			return locale.WrapError(err, "err_uninstall_confirm", "Could not confirm uninstall choice")
		}
		if !ok {
			return locale.NewInputError("err_uninstall_aborted", "Uninstall aborted by user")
		}
	}

	if err := stopServices(u.cfg, u.out, u.ipComm, params.Force); err != nil {
		return errs.Wrap(err, "Failed to stop services.")
	}

	if err := u.runUninstall(params); err != nil {
		return errs.Wrap(err, "Could not complete uninstallation")
	}

	if err := events.WaitForEvents(5*time.Second, u.an.Close, logging.Close); err != nil {
		return errs.Wrap(err, "Failed to wait for analytics and logging to close")
	}

	return nil
}
