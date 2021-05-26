package reset

import (
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
	out     output.Outputer
	auth    *authentication.Auth
	prompt  prompt.Prompter
	project *project.Project
	config  configurable
}

type configurable interface {
	CachePath() string
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
	primer.Projecter
	primer.Configurer
}

func New(prime primeable) *Reset {
	return &Reset{
		prime.Output(),
		prime.Auth(),
		prime.Prompt(),
		prime.Project(),
		prime.Config(),
	}
}

func (r *Reset) Run() error {
	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}

	confirm, err := r.prompt.Confirm("", locale.Tl("reset_confim", "You are about to reset your project to the latest commit, losing your local changes. Continue?"), new(bool))
	if err != nil {
		return locale.WrapError(err, "err_reset_confirm", "Could not confirm reset choice")
	}
	if !confirm {
		return locale.NewInputError("err_reset_aborted", "Reset aborted by user")
	}

	latestCommit, err := model.BranchCommitID(r.project.Owner(), r.project.Name(), r.project.BranchName())
	if err != nil {
		return locale.WrapError(err, "err_reset_latest_commit", "Could not get latest commit ID")
	}
	if *latestCommit == r.project.CommitUUID() {
		return locale.NewInputError("err_reset_latest", "You are already on the latest commit")
	}

	err = r.project.Source().SetCommit(latestCommit.String(), r.project.IsHeadless())
	if err != nil {
		return locale.WrapError(err, "err_reset_set_commit", "Could not update commit ID")
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.project, r.config.CachePath(), *latestCommit, false)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	r.out.Print(locale.Tl("reset_success", "Successfully reset to commit: [NOTICE]{{.V0}}[/RESET]", latestCommit.String()))

	return nil
}
