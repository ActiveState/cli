package update

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type UnlockParams struct {
	NonInteractive bool
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
	if u.project == nil {
		return rationalize.ErrNoProject
	}

	if !u.project.IsLocked() {
		u.out.Notice(locale.Tl("notice_not_locked", "The State Tool version is not locked for this project."))
		return nil
	}

	u.out.Notice(locale.Tl("unlocking_version", "Unlocking State Tool version for current project."))

	if !params.NonInteractive {
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

	err = u.cfg.Set(constants.AutoUpdateConfigKey, "true")
	if err != nil {
		return locale.WrapError(err, "err_unlock_enable_autoupdate", "Unable to re-enable automatic updates prior to unlocking")
	}

	err = projectfile.RemoveLockInfo(u.project.Source().Path())
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not update projectfile")
	}

	u.out.Notice(locale.Tl("version_unlocked", "State Tool version unlocked"))
	return nil
}

func confirmUnlock(prom prompt.Prompter) error {
	msg := locale.T("confirm_update_unlocked_version_prompt")

	defaultChoice := !prom.IsInteractive()
	confirmed, err := prom.Confirm(locale.T("confirm"), msg, &defaultChoice)
	if err != nil {
		return err
	}

	if !confirmed {
		return locale.NewInputError("err_update_lock_noconfirm", "Cancelling by your request.")
	}

	return nil
}
