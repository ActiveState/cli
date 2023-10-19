package revert

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/cmdlets/commit"
	"github.com/ActiveState/cli/pkg/localcommit"
	gqlmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Revert struct {
	out       output.Outputer
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type Params struct {
	CommitID string
	To       bool
	Force    bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Analyticer
	primer.SvcModeler
}

func New(prime primeable) *Revert {
	return &Revert{
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
		prime.Auth(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

func (r *Revert) Run(params *Params) error {
	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}
	if !strfmt.IsUUID(params.CommitID) {
		return locale.NewInputError("err_invalid_commit_id", "Invalid commit ID")
	}
	latestCommit, err := localcommit.Get(r.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	if params.CommitID == latestCommit.String() && params.To {
		return locale.NewInputError("err_revert_to_current_commit", "The commit to revert to cannot be the latest commit")
	}
	r.out.Notice(locale.Tl("operating_message", "", r.project.NamespaceString(), r.project.Dir()))
	commitID := strfmt.UUID(params.CommitID)

	var targetCommit *mono_models.Commit // the commit to revert the contents of, or the commit to revert to
	var fromCommit, toCommit strfmt.UUID
	if !params.To {
		priorCommits, err := model.CommitHistoryPaged(commitID, 0, 2)
		if err != nil {
			return errs.AddTips(
				locale.WrapError(err, "err_revert_get_commit", "", params.CommitID),
				locale.T("tip_private_project_auth"),
			)
		}
		if priorCommits.TotalCommits < 2 {
			return locale.NewInputError("err_revert_no_history", "Cannot revert commit {{.V0}}: no prior history", params.CommitID)
		}
		targetCommit = priorCommits.Commits[0]
		fromCommit = commitID
		toCommit = priorCommits.Commits[1].CommitID // parent commit
	} else {
		var err error
		targetCommit, err = model.GetCommitWithinCommitHistory(latestCommit, commitID)
		if err != nil {
			return errs.AddTips(
				locale.WrapError(err, "err_revert_get_commit", "", params.CommitID),
				locale.T("tip_private_project_auth"),
			)
		}
		fromCommit = latestCommit
		toCommit = targetCommit.CommitID
	}

	var orgs []gqlmodel.Organization
	if targetCommit.Author != nil {
		var err error
		orgs, err = model.FetchOrganizationsByIDs([]strfmt.UUID{*targetCommit.Author})
		if err != nil {
			return locale.WrapError(err, "err_revert_get_organizations", "Could not get organizations for current user")
		}
	}
	preposition := ""
	if params.To {
		preposition = " to" // need leading whitespace
	}
	if !r.out.Type().IsStructured() {
		r.out.Print(locale.Tl("revert_info", "You are about to revert{{.V0}} the following commit:", preposition))
		commit.PrintCommit(r.out, targetCommit, orgs)
	}

	defaultChoice := params.Force || !r.out.Config().Interactive
	revert, err := r.prompt.Confirm("", locale.Tl("revert_confirm", "Continue?"), &defaultChoice)
	if err != nil {
		return locale.WrapError(err, "err_revert_confirm", "Could not confirm revert choice")
	}
	if !revert {
		return locale.NewInputError("err_revert_aborted", "Revert aborted by user")
	}

	revertCommit, err := model.RevertCommitWithinHistory(fromCommit, toCommit, latestCommit)
	if err != nil {
		return errs.AddTips(
			locale.WrapError(err, "err_revert_commit", "", preposition, params.CommitID),
			locale.Tl("tip_revert_sync", "Please ensure that the local project is synchronized with the platform and that the given commit ID belongs to the current project"),
			locale.T("tip_private_project_auth"))
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, r.project, revertCommit.CommitID, true, target.TriggerRevert, r.svcModel, r.prompt)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	err = localcommit.Set(r.project.Dir(), revertCommit.CommitID.String())
	if err != nil {
		return errs.Wrap(err, "Unable to set local commit")
	}

	r.out.Print(output.Prepare(
		locale.Tl("revert_success", "Successfully reverted{{.V0}} commit: {{.V1}}", preposition, params.CommitID),
		&struct {
			CurrentCommitID string `json:"current_commit_id"`
		}{
			revertCommit.CommitID.String(),
		},
	))
	r.out.Notice(locale.T("operation_success_local"))
	return nil
}

func containsCommitID(history []*mono_models.Commit, commitID strfmt.UUID) bool {
	for _, c := range history {
		if c.CommitID == commitID {
			return true
		}
	}
	return false
}
