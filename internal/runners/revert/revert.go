package revert

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/commit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Revert struct {
	out     output.Outputer
	prompt  prompt.Prompter
	project *project.Project
}

type Params struct {
	CommitID string
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
}

func New(prime primeable) *Revert {
	return &Revert{
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
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
	commitID := strfmt.UUID(params.CommitID)
	revertCommit, err := model.GetCommit(strfmt.UUID(params.CommitID))
	if err != nil {
		return locale.WrapError(err, "err_revert_get_commit", "Could not fetch commit details for commit with ID: {{.V0}}", params.CommitID)
	}

	orgs, fail := model.FetchOrganizationsByIDs([]strfmt.UUID{revertCommit.Author})
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_revert_get_organizations", "Could not get organizations for current user")
	}
	commit.PrintCommit(r.out, revertCommit, orgs)

	revert, fail := r.prompt.Confirm("", locale.Tl("revert_confirm", "Revert to commit: {{.V0}}?", params.CommitID), false)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_revert_confirm", "Could not confirm revert choice")
	}
	if !revert {
		return nil
	}

	err = model.RevertCommit(r.project.Owner(), r.project.Name(), r.project.CommitUUID(), commitID)
	if err != nil {
		return locale.WrapError(
			err,
			"err_revert_commit",
			"Could not revert to commit: {{.V0}} please ensure that the local project is synchronized with the platform and that the given commit ID belongs to the current project",
			params.CommitID,
		)
	}

	r.out.Print(locale.Tl("revert_success", "Sucessfully reverted to commit: {{.V0}}", params.CommitID))
	r.out.Print(locale.T("update_config"))
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
