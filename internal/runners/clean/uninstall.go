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
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/svcctl"
)

type Uninstall struct {
	out    output.Outputer
	prompt prompt.Prompter
	cfg    *config.Instance
	ipComm svcctl.IPCommunicator
	an     analytics.Dispatcher
}

type UninstallParams struct {
	Force  bool
	All    bool
	Prompt bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Configurer
	primer.IPCommunicator
	primer.Analyticer
	primer.SvcModeler
}

func NewUninstall(prime primeable) (*Uninstall, error) {
	return newUninstall(prime.Output(), prime.Prompt(), prime.Config(), prime.IPComm(), prime.Analytics())
}

func newUninstall(out output.Outputer, prompt prompt.Prompter, cfg *config.Instance, ipComm svcctl.IPCommunicator, an analytics.Dispatcher) (*Uninstall, error) {
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
	} else {
		defaultChoice := !u.prompt.IsInteractive()
		confirmMessage := locale.T("uninstall_confirm")
		if params.All {
			confirmMessage = locale.T("uninstall_confirm_all")
		}
		ok, kind, err := u.prompt.Confirm(locale.T("confirm"), confirmMessage, &defaultChoice, ptr.To(true))
		if err != nil {
			return errs.Wrap(err, "Unable to confirm")
		}
		if !ok {
			return locale.NewInputError("err_uninstall_aborted", "Uninstall aborted by user")
		}
		switch kind {
		case prompt.NonInteractive:
			u.out.Notice(locale.T("prompt_continue_non_interactive"))
		case prompt.Force:
			u.out.Notice(locale.T("prompt_continue_force"))
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
