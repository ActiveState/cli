package branch

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Add struct {
	out     output.Outputer
	project *project.Project
}

type AddParams struct {
	Label string
}

func NewAdd(prime primeable) *Add {
	return &Add{
		out:     prime.Output(),
		project: prime.Project(),
	}
}

func (a *Add) Run(params AddParams) error {
	logging.Debug("ExecuteAdd")

	project, err := model.FetchProjectByName(a.project.Owner(), a.project.Name())
	if err != nil {
		return locale.WrapError(err, "err_fetch_project", a.project.Namespace().String())
	}

	branchID, err := model.AddBranch(project.ProjectID, params.Label)
	if err != nil {
		return locale.WrapError(err, "err_add_branch", "Could not add branch")
	}

	localBranch := a.project.BranchName()
	branch, err := model.BranchForProjectByName(project, localBranch)
	if err != nil {
		return locale.WrapError(err, "err_fetch_branch", "", localBranch)
	}

	err = model.UpdateBranchTracking(branchID, a.project.CommitUUID(), branch.BranchID, model.TrackingIgnore)
	if err != nil {
		return locale.WrapError(err, "err_add_branch_update_tracking", "Could not update branch: {{.V0}} with tracking information", params.Label)
	}

	a.out.Print(locale.Tl("branch_add_success", "Successfully added branch: {{.V0}}", params.Label))

	return nil
}
