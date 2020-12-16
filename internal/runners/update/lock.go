package update

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type LockParams struct {
	Force bool
}

type Lock struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
}

func NewLock(prime primeable) *Lock {
	return &Lock{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (l *Lock) Run(params *LockParams) error {
	if l.project.IsLocked() && !params.Force {
		if err := confirmLock(l.prompt); err != nil {
			return locale.WrapError(err, "err_update_lock_confirm", "Could not confirm whether to update.")
		}
	}
	l.out.Notice(locale.Tl("locking_version", "Locking State Tool to the current version."))

	err := projectfile.AddLockInfo(l.project.Source().Path(), constants.BranchName, constants.Version)
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not update projectfile")
	}

	l.out.Print(locale.Tl("version_locked", "Version locked at {{.V0}}", constants.Version))
	return nil
}

func confirmLock(prom prompt.Prompter) error {
	msg := locale.T("confirm_update_locked_version_prompt")

	confirmed, err := prom.Confirm(locale.T("confirm"), msg, false)
	if err != nil {
		return err
	}

	if !confirmed {
		return locale.NewInputError("err_update_lock_noconfirm", "Cancelling by your request.")
	}

	return nil
}
