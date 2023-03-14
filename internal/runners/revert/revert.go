package revert

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/cmdlets/commit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"

	gqlmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
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

type commitDetails struct {
	Date        string
	Author      string
	Description string
	Changeset   []changeset `locale:"changeset,Changes"`
}

type changeset struct {
	Operation   string `locale:"operation,Operation"`
	Requirement string `locale:"requirement,Requirement"`
}

func (r *Revert) Run(params *Params) error {
	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}
	if !strfmt.IsUUID(params.CommitID) {
		return locale.NewInputError("err_invalid_commit_id", "Invalid commit ID")
	}
	if params.CommitID == r.project.CommitID() && params.To {
		return locale.NewInputError("err_revert_to_current_commit", "Cannot revert to current commit")
	}
	r.out.Notice(locale.Tl("operating_message", "", r.project.NamespaceString(), r.project.Dir()))
	commitID := strfmt.UUID(params.CommitID)

	var targetCommit *mono_models.Commit
	var fromCommit, toCommit strfmt.UUID
	latestCommit := r.project.CommitUUID()
	if !params.To {
		priorCommits, err := model.CommitHistoryPaged(commitID, 0, 2)
		if err != nil {
			return locale.WrapError(err, "err_revert_get_commit", "Could not fetch commit details for commit with ID: {{.V0}}", params.CommitID)
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
			return locale.WrapError(err, "err_revert_to_get_commit", "Could not fetch commit details for commit with ID: {{.V0}}", params.CommitID)
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
	r.out.Print(locale.Tl("revert_info", "You are about to revert{{.V0}} the following commit:", preposition))
	commit.PrintCommit(r.out, targetCommit, orgs)

	defaultChoice := params.Force
	revert, err := r.prompt.Confirm("", locale.Tl("revert_confirm", "Continue?"), &defaultChoice)
	if err != nil {
		return locale.WrapError(err, "err_revert_confirm", "Could not confirm revert choice")
	}
	if !revert {
		return locale.NewInputError("err_revert_aborted", "Revert aborted by user")
	}

	revertCommit, err := model.RevertCommitWithinHistory(fromCommit, toCommit, latestCommit)
	if err != nil {
		return locale.WrapError(
			err,
			"err_revert_commit",
			"Could not revert{{.V0}} commit: {{.V1}} please ensure that the local project is synchronized with the platform and that the given commit ID belongs to the current project",
			preposition,
			params.CommitID,
		)
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, r.project, revertCommit.CommitID, true, target.TriggerRevert, r.svcModel)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	err = r.project.SetCommit(revertCommit.CommitID.String())
	if err != nil {
		return locale.WrapError(err, "err_revert_set_commit", "Could not set revert commit ID in projectfile")
	}

	r.out.Print(locale.Tl("revert_success", "Successfully reverted{{.V0}} commit: {{.V1}}", preposition, params.CommitID))
	r.out.Print(locale.T("operation_success_local"))
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
