package buildplanner

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

type StageCommitRequirement struct {
	Name      string
	Version   []types.VersionRequirement
	Namespace string
	Revision  *int
	Operation types.Operation
}

type StageCommitParams struct {
	Owner        string
	Project      string
	ParentCommit string
	Description  string
	// Commits can have either requirements (e.g. installing a package)...
	Requirements []StageCommitRequirement
	// ... or commits can have an expression (e.g. from pull). When pulling an expression, we do not
	// compute its changes into a series of above operations. Instead, we just pass the new
	// expression directly.
	Script *buildscript.BuildScript
}

func (b *BuildPlanner) StageCommit(params StageCommitParams) (strfmt.UUID, error) {
	logging.Debug("StageCommit, params: %+v", params)
	script := params.Script
	if script == nil {
		var err error
		script, err = b.GetBuildScript(params.ParentCommit)
		if err != nil {
			return "", errs.Wrap(err, "Failed to get build expression")
		}

		var containsPackageOperation bool
		for _, req := range params.Requirements {
			if req.Namespace == types.NamespacePlatform {
				err = script.UpdatePlatform(req.Operation, strfmt.UUID(req.Name))
				if err != nil {
					return "", errs.Wrap(err, "Failed to update build expression with platform")
				}
			} else {
				requirement := types.Requirement{
					Namespace:          req.Namespace,
					Name:               req.Name,
					VersionRequirement: req.Version,
					Revision:           req.Revision,
				}

				err = script.UpdateRequirement(req.Operation, requirement)
				if err != nil {
					return "", errs.Wrap(err, "Failed to update build expression with requirement")
				}
				containsPackageOperation = true
			}
		}

		if containsPackageOperation {
			if err := script.SetDefaultAtTime(); err != nil {
				return "", errs.Wrap(err, "Failed to set default timestamp")
			}
		}

	}

	exprB, err := script.MarshalBuildExpression()
	if err != nil {
		return "", errs.Wrap(err, "Failed to marshal build expression")
	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, params.Description, script.AtTime(), exprB)
	resp := &response.StageCommitResult{}
	if err := b.client.Run(request, resp); err != nil {
		return "", processBuildPlannerError(err, "failed to stage commit")
	}

	if resp.Commit == nil {
		return "", errs.New("Staged commit is nil")
	}

	if response.IsErrorResponse(resp.Commit.Type) {
		return "", response.ProcessCommitError(resp.Commit, "Could not process error response from stage commit")
	}

	if resp.Commit.CommitID == "" {
		return "", errs.New("Staged commit does not contain commitID")
	}

	return resp.Commit.CommitID, nil
}

func (b *BuildPlanner) RevertCommit(organization, project, parentCommitID, commitID string) (strfmt.UUID, error) {
	logging.Debug("RevertCommit, organization: %s, project: %s, commitID: %s", organization, project, commitID)
	resp := &response.RevertCommitResult{}
	err := b.client.Run(request.RevertCommit(organization, project, parentCommitID, commitID), resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to revert commit")
	}

	if resp.RevertedCommit == nil {
		return "", errs.New("Revert commit response is nil")
	}

	if response.IsErrorResponse(resp.RevertedCommit.Type) {
		return "", response.ProcessRevertCommitError(resp.RevertedCommit, "Could not revert commit")
	}

	if resp.RevertedCommit.Commit == nil {
		return "", errs.New("Revert commit's commit is nil'")
	}

	if response.IsErrorResponse(resp.RevertedCommit.Commit.Type) {
		return "", response.ProcessCommitError(resp.RevertedCommit.Commit, "Could not process error response from revert commit")
	}

	if resp.RevertedCommit.Commit.CommitID == "" {
		return "", errs.New("Commit does not contain commitID")
	}

	return resp.RevertedCommit.Commit.CommitID, nil
}

type MergeCommitParams struct {
	Owner     string
	Project   string
	TargetRef string // the commit ID or branch name to merge into
	OtherRef  string // the commit ID or branch name to merge from
	Strategy  types.MergeStrategy
}

func (b *BuildPlanner) MergeCommit(params *MergeCommitParams) (strfmt.UUID, error) {
	logging.Debug("MergeCommit, owner: %s, project: %s", params.Owner, params.Project)
	request := request.MergeCommit(params.Owner, params.Project, params.TargetRef, params.OtherRef, params.Strategy)
	resp := &response.MergeCommitResult{}
	err := b.client.Run(request, resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to merge commit")
	}

	if resp.MergedCommit == nil {
		return "", errs.New("MergedCommit is nil")
	}

	if response.IsErrorResponse(resp.MergedCommit.Type) {
		return "", response.ProcessMergedCommitError(resp.MergedCommit, "Could not merge commit")
	}

	if resp.MergedCommit.Commit == nil {
		return "", errs.New("Merge commit's commit is nil'")
	}

	if response.IsErrorResponse(resp.MergedCommit.Commit.Type) {
		return "", response.ProcessCommitError(resp.MergedCommit.Commit, "Could not process error response from merge commit")
	}

	return resp.MergedCommit.Commit.CommitID, nil
}
