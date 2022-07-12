package revert

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/installation/storage"
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
	commitID := strfmt.UUID(params.CommitID)
	revertToCommit, err := model.GetCommit(commitID)
	if err != nil {
		return locale.WrapError(err, "err_revert_get_commit", "Could not fetch commit details for commit with ID: {{.V0}}", params.CommitID)
	}

	var orgs []gqlmodel.Organization
	if revertToCommit.Author != nil {
		orgs, err = model.FetchOrganizationsByIDs([]strfmt.UUID{*revertToCommit.Author})
		if err != nil {
			return locale.WrapError(err, "err_revert_get_organizations", "Could not get organizations for current user")
		}
	}
	r.out.Print(locale.Tl("revert_info", "You are about to revert to the following commit:"))
	commit.PrintCommit(r.out, revertToCommit, orgs)

	revert, err := r.prompt.Confirm("", locale.Tl("revert_confirm", "Continue?"), new(bool))
	if err != nil {
		return locale.WrapError(err, "err_revert_confirm", "Could not confirm revert choice")
	}
	if !revert {
		return nil
	}

	revertCommit, err := model.RevertCommit(r.project.CommitUUID(), commitID)
	if err != nil {
		return locale.WrapError(
			err,
			"err_revert_commit",
			"Could not revert to commit: {{.V0}} please ensure that the local project is synchronized with the platform and that the given commit ID belongs to the current project",
			params.CommitID,
		)
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, r.project, storage.CachePath(), revertCommit.CommitID, true, target.TriggerRevert, r.svcModel)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	err = r.project.SetCommit(revertCommit.CommitID.String())
	if err != nil {
		return locale.WrapError(err, "err_revert_set_commit", "Could not set revert commit ID in projectfile")
	}

	r.out.Print(locale.Tl("revert_success", "Successfully reverted to commit: {{.V0}}", params.CommitID))
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
