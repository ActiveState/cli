package use

import (
	"runtime"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
)

type confirmAble interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
}

type Reset struct {
	confirm  confirmAble
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

	if !params.Force {
		ok, err := u.confirm.Confirm(locale.T("confirm"),
			locale.Tl("clean_config_confirm", "You are about to reset your default project runtime. Continue?"), new(bool))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	reset, err := globaldefault.ResetDefaultActivation(u.subshell, u.config)
	if err != nil {
		return locale.WrapError(err, "err_use_reset", "Could not reset your global default project.")
	} else if !reset {
		u.out.Notice(locale.Tl("use_reset_notice_not_reset", "No global default project to reset"))
		return nil
	}

	u.out.Notice(locale.Tl("use_reset_notice_reset", "Reset default project runtime"))

	if runtime.GOOS == "windows" {
		u.out.Notice(locale.T("use_reset_notice_windows"))
	} else {
		u.out.Notice(locale.T("use_reset_notice"))
	}

	return nil
}
