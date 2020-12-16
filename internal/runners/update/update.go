package update

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
)

type Update struct {
	project *project.Project
	out     output.Outputer
	prompt  prompt.Prompter
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Prompter
}

func New(prime primeable) *Update {
	return &Update{
		prime.Project(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (u *Update) Run() error {
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

	if err = up.Run(u.out, false); err != nil {
		if os.IsPermission(errs.InnerError(err)) {
			return locale.WrapError(err, "err_update_failed_due_to_permissions", "Update failed due to permission error.  You may have to re-run the command as a privileged user.")
		}
		return locale.WrapError(err, "err_update_failed", "Update failed, please try again later or try reinstalling the State Tool.")
	}

	u.out.Print(locale.Tl("version_updated", "Version updated to {{.V0}}@{{.V1}}", constants.BranchName, info.Version))
	return nil
}
