package update

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type UnlockParams struct {
	Force bool
}

type Unlock struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
	cfg     updater.Configurable
}

func NewUnlock(prime primeable) *Unlock {
	return &Unlock{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
		prime.Config(),
	}
}

func (u *Unlock) Run(params *UnlockParams) error {
	if !u.project.IsLocked() {
		u.out.Notice(locale.Tl("notice_not_locked", "The State Tool version is not locked for this project."))
		return nil
	}

	u.out.Notice(locale.Tl("unlocking_version", "Unlocking State Tool version for current project."))

	if !params.Force {
		err := confirmUnlock(u.prompt)
		if err != nil {
			return locale.WrapError(err, "err_update_unlock_confirm", "Unlock cancelled by user.")
		}
	}

	// Invalidate the installer version lock.
	err := u.cfg.Set(updater.CfgKeyInstallVersion, "")
	if err != nil {
		multilog.Error("Failed to invalidate installer version lock on `state update lock` invocation: %v", err)
	}

	err = projectfile.RemoveLockInfo(u.project.Source().Path())
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not update projectfile")
	}

	u.out.Print(locale.Tl("version_unlocked", "State Tool version unlocked"))
	return nil
}

func confirmUnlock(prom prompt.Prompter) error {
	msg := locale.T("confirm_update_unlocked_version_prompt")

	confirmed, err := prom.Confirm(locale.T("confirm"), msg, new(bool))
	if err != nil {
		return err
	}

	if !confirmed {
		return locale.NewInputError("err_update_lock_noconfirm", "Cancelling by your request.")
	}

	return nil
}
