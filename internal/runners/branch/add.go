package branch

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
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
		return locale.WrapError(err, "branch_list_proejct_err", "Could not get project details for project: {{.V0}}", a.project.Namespace().String())
	}

	branchID, err := model.AddBranch(project.ProjectID, params.Label)
	if err != nil {
		return locale.WrapError(err, "err_add_branch", "Could not add branch")
	}

	var trackingID *strfmt.UUID
	for _, branch := range project.Branches {
		if branch.Default {
			trackingID = &branch.BranchID
		}
	}
	if trackingID == nil {
		return locale.NewError("err_add_branch_no_default", "Could not determine default branch")
	}

	err = model.UpdateBranchTracking(*branchID, *trackingID)
	if err != nil {
		logging.Debug("Unable to update tracking information, attempting to delete branch")
		derr := model.DeleteBranch(*branchID)
		if err != nil {
			logging.Debug("Could not delete branch %s, got error: %v", params.Label, derr)
		}
		return locale.WrapError(err, "err_add_branch_update_tracking", "Could not update branch: {{.V0}} with tracking information", params.Label)
	}

	a.out.Print(locale.Tl("branch_add_success", "Successfully added branch: {{.V0}}", params.Label))

	return nil
}
