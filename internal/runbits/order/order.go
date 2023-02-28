package order

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/orderfile"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"gopkg.in/yaml.v2"
)

type CheckParams struct {
	Path    string
	Project *project.Project
	Out     output.Outputer
	Auth    *authentication.Auth
}

func Check(params *CheckParams) (strfmt.UUID, error) {
	resultCommitID := strfmt.UUID(params.Project.CommitID())
	of, err := orderfile.FromPath(params.Path)
	if err != nil {
		return "", errs.Wrap(err, "Could not read order file")
	}

	bp := model.NewBuildPlanner(params.Auth)
	script, err := bp.GetBuildScript(params.Project.Owner(), params.Project.Name(), params.Project.CommitID())
	if err != nil {
		return "", errs.Wrap(err, "Could not get build script")
	}

	if of.Script().Equals(script) {
		return resultCommitID, nil
	}

	params.Out.Notice(locale.Tl("orderfile_outdated", "Order file is outdated"))
	params.Out.Print(locale.Tl("orderfile_update", "Updating project to reflect order file changes..."))

	var commitExists bool
	_, err = platformModel.GetCommit(strfmt.UUID(params.Project.CommitID()))
	if err != nil {
		if !errs.Matches(err, &platformModel.ErrCommitNotFound{}) {
			return "", locale.WrapError(err, "err_orderfile_update", "Could not get commit details.")
		}
		commitExists = true
	}

	if !commitExists {
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

		// TODO: Don't like this side effect here. Is there another way?
		if err := params.Project.SetCommit(commit.CommitID); err != nil {
			return "", locale.WrapError(err, "err_orderfile_update", "Could not update project file commit ID.")
		}

		script, err = bp.GetBuildScript(params.Project.Owner(), params.Project.Name(), commit.CommitID)
		if err != nil {
			return "", errs.Wrap(err, "Could not get build script")
		}
		resultCommitID = strfmt.UUID(commit.CommitID)
	}

	data, err := yaml.Marshal(script)
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal build script")
	}
	logging.Debug("New order file: %s", string(data))
	if err := of.Update(script); err != nil {
		return "", locale.WrapError(err, "err_orderfile_update", "Could not update order file.")
	}

	return resultCommitID, nil
}
