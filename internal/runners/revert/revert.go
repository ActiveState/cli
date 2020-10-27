package revert

import (
	"fmt"
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
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

func (r *Revert) Run(params *Params) error {
	commitID := strfmt.UUID(params.CommitID)
	commit, err := model.GetCommit(strfmt.UUID(params.CommitID))
	if err != nil {
		return err
	}

	history, fail := model.CommitHistory(r.project.Owner(), r.project.Name())
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_revert_get_history", "Could not get project commit history")
	}

	if !containsCommitID(history, commitID) {
		return locale.NewError("err_revert_invalid_commit_id", "Commit ID: {{.V0}} does not belong to the project: {{.V1}}", params.CommitID, r.project.Namespace().String())
	}

	count, fail := model.CommitsBehindLatest(r.project.Owner(), r.project.Name(), r.project.CommitID())
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_revert_commits_behind", "Could not determine if local project is synchronized with platform")
	}
	if count > 0 {
		return locale.NewInputError("err_revert_behind_latest", "Your project is {{.V0}} commits behind. Please run `state pull` to syncronize your project and run `state revert` again", strconv.Itoa(count))
	}

	// TODO: Create a method/type/map for handling output
	r.out.Print(fmt.Sprintf("%s by %s", commit.AtTime.String(), commit.Username))
	r.out.Print(fmt.Sprintf("Description: %s", commit.Message))
	r.out.Print("Changes made:")
	for _, c := range commit.Changeset {
		// The requirement does not print well when it is a platform (prints uuid)
		r.out.Print(fmt.Sprintf("  %s %s", c.Operation, c.Requirement))
	}

	revert, fail := r.prompt.Confirm(locale.Tl("revert_confirm", fmt.Sprintf("Revert to commit: %s", params.CommitID)), false)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_revert_confirm", "Could not confirm revert choice")
	}
	if !revert {
		return nil
	}

	err = model.RevertCommit(r.project.Owner(), r.project.Name(), r.project.CommitUUID(), commitID)
	if err != nil {
		return locale.WrapError(err, "err_revert_commit", "Could not revert to commit: {{.V0}}", params.CommitID)
	}

	r.out.Print(locale.Tl("revert_success", "Sucessfully reverted to commit: {{.V0}}"))
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
