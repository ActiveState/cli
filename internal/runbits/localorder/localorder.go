package localorder

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/orderfile"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type CheckParams struct {
	Path         string
	Project      *project.Project
	CustomCommit string
	Out          output.Outputer
	Auth         *authentication.Auth
}

func Check(params *CheckParams) (strfmt.UUID, error) {
	commitID := strfmt.UUID(params.Project.CommitID())
	if params.CustomCommit != "" {
		commitID = strfmt.UUID(params.CustomCommit)
	}

	of, err := orderfile.FromPath(params.Path)
	if err != nil {
		// TODO: Should we be checking for this error here?
		if orderfile.IsErrOrderFileDoesNotExist(err) {
			return commitID, nil
		}
		return "", errs.Wrap(err, "Could not read order file")
	}

	bp := model.NewBuildPlanner(params.Auth)
	script, err := bp.GetBuildScript(params.Project.Owner(), params.Project.Name(), commitID.String())
	if err != nil {
		return "", errs.Wrap(err, "Could not get build script")
	}

	if of.Script().Equals(script) {
		return commitID, nil
	}

	params.Out.Notice(locale.Tl("orderfile_outdated", "Order file is outdated"))
	params.Out.Print(locale.Tl("orderfile_update", "Updating project to reflect order file changes..."))

	commit, err := bp.PushCommit(model.PushCommitParams{
		Owner:        params.Project.Owner(),
		Project:      params.Project.Name(),
		ParentCommit: params.Project.CommitID(),
		BranchRef:    params.Project.BranchName(),
		Description:  locale.Tl("orderfile_commit_description", "Update project due to local oder file change."),
		Script:       of.Script(),
	})
	if err != nil {
		return "", locale.WrapError(err, "err_orderfile_update", "Could not update project to reflect order file changes.")
	}
	commitID = strfmt.UUID(commit.CommitID)

	// TODO: Don't like this side effect here. Is there another way?
	if err := params.Project.SetCommit(commitID.String()); err != nil {
		return "", locale.WrapError(err, "err_orderfile_update", "Could not update project file commit ID.")
	}

	script, err = bp.GetBuildScript(params.Project.Owner(), params.Project.Name(), commitID.String())
	if err != nil {
		return "", errs.Wrap(err, "Could not get build script")
	}

	if err := of.Update(script); err != nil {
		return "", locale.WrapError(err, "err_orderfile_update", "Could not update order file.")
	}

	return commitID, nil
}
