package branch

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	//"github.com/ActiveState/cli/pkg/localcommit" // re-enable in DX-2307
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

	if a.project == nil {
		return locale.NewInputError("err_no_project")
	}

	project, err := model.LegacyFetchProjectByName(a.project.Owner(), a.project.Name())
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

	commitID, err := commitmediator.Get(a.project)
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}

	err = model.UpdateBranchTracking(branchID, commitID, branch.BranchID, model.TrackingIgnore)
	if err != nil {
		return locale.WrapError(err, "err_add_branch_update_tracking", "Could not update branch: {{.V0}} with tracking information", params.Label)
	}

	a.out.Print(output.Prepare(
		locale.Tl("branch_add_success", "Successfully added branch: {{.V0}}", params.Label),
		&struct {
			Branch string `json:"branch"`
		}{params.Label},
	))

	return nil
}
