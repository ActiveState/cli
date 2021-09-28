package reset

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Reset struct {
	out       output.Outputer
	auth      *authentication.Auth
	prompt    prompt.Prompter
	project   *project.Project
	analytics analytics.AnalyticsDispatcher
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
	primer.Projecter
	primer.Configurer
	primer.Analyticer
}

func New(prime primeable) *Reset {
	return &Reset{
		prime.Output(),
		prime.Auth(),
		prime.Prompt(),
		prime.Project(),
		prime.Analytics(),
	}
}

func (r *Reset) Run() error {
	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}

	latestCommit, err := model.BranchCommitID(r.project.Owner(), r.project.Name(), r.project.BranchName())
	if err != nil {
		return locale.WrapError(err, "err_reset_latest_commit", "Could not get latest commit ID")
	}
	if *latestCommit == r.project.CommitUUID() {
		return locale.NewInputError("err_reset_latest", "You are already on the latest commit")
	}

	r.out.Print(locale.Tl("reset_commit", "Your project will be reset to [ACTIONABLE]{{.V0}}[/RESET]\n", latestCommit.String()))

	confirm, err := r.prompt.Confirm("", locale.Tl("reset_confim", "Resetting is destructive, you will lose any changes that were not pushed. Are you sure you want to do this?"), new(bool))
	if err != nil {
		return locale.WrapError(err, "err_reset_confirm", "Could not confirm reset choice")
	}
	if !confirm {
		return locale.NewInputError("err_reset_aborted", "Reset aborted by user")
	}

	err = r.project.SetCommit(latestCommit.String())
	if err != nil {
		return locale.WrapError(err, "err_reset_set_commit", "Could not update commit ID")
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, r.project, storage.CachePath(), *latestCommit, true)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	r.out.Print(locale.Tl("reset_success", "Successfully reset to commit: [NOTICE]{{.V0}}[/RESET]", latestCommit.String()))

	return nil
}
