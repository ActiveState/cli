package reset

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Params struct {
	Force    bool
	CommitID string
}

type Reset struct {
	out       output.Outputer
	auth      *authentication.Auth
	prompt    prompt.Prompter
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
	primer.Projecter
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

func New(prime primeable) *Reset {
	return &Reset{
		prime.Output(),
		prime.Auth(),
		prime.Prompt(),
		prime.Project(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

type outputFormat struct {
	message  string
	CommitID string `json:"commitID"`
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	return f.message
}

func (f *outputFormat) MarshalStructured(format output.Format) interface{} {
	return f
}

func (r *Reset) Run(params *Params) error {
	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}
	r.out.Notice(locale.Tl("operating_message", "", r.project.NamespaceString(), r.project.Dir()))

	var commitID strfmt.UUID
	if params.CommitID == "" {
		latestCommit, err := model.BranchCommitID(r.project.Owner(), r.project.Name(), r.project.BranchName())
		if err != nil {
			return locale.WrapError(err, "err_reset_latest_commit", "Could not get latest commit ID")
		}
		if *latestCommit == r.project.CommitUUID() {
			return locale.NewInputError("err_reset_latest", "You are already on the latest commit")
		}
		commitID = *latestCommit
	} else {
		if !strfmt.IsUUID(params.CommitID) {
			return locale.NewInputError("Invalid commit ID")
		}
		commitID = strfmt.UUID(params.CommitID)
		if commitID == r.project.CommitUUID() {
			return locale.NewInputError("err_reset_same_commitid", "Your project is already at the given commit ID")
		}
		history, err := model.CommitHistoryFromID(commitID)
		if err != nil || len(history) == 0 {
			return locale.WrapInputError(err, "err_reset_commitid", "The given commit ID does not exist")
		}
	}

	r.out.Notice(locale.Tl("reset_commit", "Your project will be reset to [ACTIONABLE]{{.V0}}[/RESET]\n", commitID.String()))

	defaultChoice := params.Force
	confirm, err := r.prompt.Confirm("", locale.Tl("reset_confim", "Resetting is destructive, you will lose any changes that were not pushed. Are you sure you want to do this?"), &defaultChoice)
	if err != nil {
		return locale.WrapError(err, "err_reset_confirm", "Could not confirm reset choice")
	}
	if !confirm {
		return locale.NewInputError("err_reset_aborted", "Reset aborted by user")
	}

	err = r.project.SetCommit(commitID.String())
	if err != nil {
		return locale.WrapError(err, "err_reset_set_commit", "Could not update commit ID")
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, r.project, commitID, true, target.TriggerReset, r.svcModel)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	r.out.Print(&outputFormat{
		locale.Tl("reset_success", "Successfully reset to commit: [NOTICE]{{.V0}}[/RESET]", commitID.String()),
		commitID.String(),
	})

	return nil
}
