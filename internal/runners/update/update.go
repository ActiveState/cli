package update

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Params struct {
	Lock  bool
	Force bool
}

type Update struct {
	project *project.Project
	out     output.Outputer
}

func New(pj *project.Project, out output.Outputer) *Update {
	return &Update{
		pj,
		out,
	}
}

func (u *Update) Run(params *Params) error {
	return run(params.Lock, isLocked(), params.Force, u.runLock, u.runUpdateLock, u.runUpdateGlobal, confirmUpdateLock)
}

func run(lock, isLocked, force bool, runLock, runUpdateLock, runUpdateGlobal, confirmLock func() error) error {
	if lock {
		return runLock()
	}
	if !lock && isLocked {
		if !force {
			if err := confirmLock(); err != nil {
				return locale.WrapError(err, "err_update_lock_confirm", "Could not confirm whether to update.")
			}
		}
		return runUpdateLock()
	}
	return runUpdateGlobal()
}

func (u *Update) runLock() error {
	u.out.Notice(locale.Tl("locking_version", "Locking State Tool to the current version."))

	if u.project.Version() != "" {
		u.out.Print(locale.Tl("lock_project_uptodate", "Your project is already locked, did you mean to run 'state update' (without the --lock flag)?"))
		return nil
	}

	pj := u.project.Source()
	pj.Branch = constants.BranchName
	pj.Version = constants.Version

	if fail := pj.Save(); fail != nil {
		return locale.WrapError(fail, "err_update_save", "Failed to update your activestate.yaml with the new version.")
	}

	u.out.Print(locale.Tl("version_locked", "Version locked at {{.V0}}", constants.Version))
	return nil
}

func (u *Update) runUpdateLock() error {
	u.out.Notice(locale.Tl("updating_lock_version", "Locking State Tool to latest version available for your project."))

	info, err := updater.New(u.project.Version()).Info()
	if err != nil {
		return locale.WrapError(err, "err_update_updater", "Could not retrieve update information.")
	}

	if info == nil {
		u.out.Print(locale.Tl("update_project_uptodate", "Your project is already using the latest State Tool version available."))
		return nil
	}

	pj := u.project.Source()
	pj.Branch = constants.BranchName
	pj.Version = info.Version

	if fail := pj.Save(); fail != nil {
		return locale.WrapError(fail, "err_update_save", "Failed to update your activestate.yaml with the new version.")
	}

	u.out.Print(locale.Tl("version_lock_updated", "Locked version updated to {{.V0}}", constants.Version))
	return nil
}

func (u *Update) runUpdateGlobal() error {
	u.out.Notice(locale.Tl("updating_version", "Updating State Tool to latest version available."))

	up := updater.New(constants.Version)
	info, err := up.Info()
	if err != nil {
		return locale.WrapError(err, "err_update_updater", "Could not retrieve update information.")
	}

	if info == nil {
		u.out.Print(locale.Tl("update_uptodate", "You are already using the latest State Tool version available."))
		return nil
	}

	if err = up.Run(u.out); err != nil {
		failures.Handle(err, locale.T("err_update_failed"))
		return locale.WrapError(err, "err_update_failed", "Update failed, please try again later or try reinstalling the State Tool.")
	}

	u.out.Print(locale.Tl("version_updated", "Version updated to {{.V0}}", constants.Version))
	return nil
}

func confirmUpdateLock() error {
	msg := locale.T("confirm_update_locked_version_prompt")

	prom := prompt.New()
	confirmed, fail := prom.Confirm(msg, false)
	if fail != nil {
		return fail.ToError()
	}

	if !confirmed {
		return locale.NewInputError("err_update_lock_noconfirm", "Cancelling by your request.")
	}

	return nil
}

func isLocked() bool {
	pj, fail := projectfile.GetSafe()
	return fail == nil && pj.Version != ""
}
