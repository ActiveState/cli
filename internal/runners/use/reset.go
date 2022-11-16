package use

import (
	"runtime"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
)

type Reset struct {
	prompt   prompt.Prompter
	out      output.Outputer
	config   *config.Instance
	subshell subshell.SubShell
}

type ResetParams struct {
	Force bool
}

func NewReset(prime primeable) *Reset {
	return &Reset{
		prime.Prompt(),
		prime.Output(),
		prime.Config(),
		prime.Subshell(),
	}
}

func (u *Reset) Run(params *ResetParams) error {
	logging.Debug("Resetting default project runtime")

	if !globaldefault.IsSet(u.config) {
		u.out.Notice(locale.T("use_reset_notice_not_reset"))
    return nil
	}

	defaultChoice := params.Force
	ok, err := u.prompt.Confirm(locale.T("confirm"),
		locale.Tl("use_reset_confirm", "You are about to stop using your project runtime. Continue?"), &defaultChoice)
	if err != nil {
		return err
	}
	if !ok {
		return locale.NewInputError("err_reset_aborted", "Reset aborted by user")
	}

	reset, err := globaldefault.ResetDefaultActivation(u.subshell, u.config)
	if err != nil {
		return locale.WrapError(err, "err_use_reset", "Could not stop using your project.")
	} else if !reset {
		u.out.Notice(locale.Tl("use_reset_notice_not_reset", "No project to stop using"))
		return nil
	}

	u.out.Notice(locale.Tl("use_reset_notice_reset", "Stopped using your project runtime"))

	if runtime.GOOS == "windows" {
		u.out.Notice(locale.T("use_reset_notice_windows"))
	} else {
		u.out.Notice(locale.T("use_reset_notice"))
	}

	return nil
}
