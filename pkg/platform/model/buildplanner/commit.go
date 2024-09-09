package buildplanner

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
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
	Script       *buildscript.BuildScript
}

func (b *BuildPlanner) StageCommit(params StageCommitParams) (*Commit, error) {
	logging.Debug("StageCommit, params: %+v", params)
	script := params.Script

	if script == nil {
		return nil, errs.New("Script is nil")
	}

	expression, err := script.MarshalBuildExpression()
	if err != nil {
		return nil, errs.Wrap(err, "Failed to marshal build expression")
	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, params.Description, ptr.To(script.AtTime()), expression)
	resp := &response.StageCommitResult{}
	if err := b.client.Run(request, resp); err != nil {
		return nil, processBuildPlannerError(err, "failed to stage commit")
	}

	if resp.Commit == nil {
		return nil, errs.New("Staged commit is nil")
	}

	if response.IsErrorResponse(resp.Commit.Type) {
		return nil, response.ProcessCommitError(resp.Commit, "Could not process error response from stage commit")
	}

	if resp.Commit.CommitID == "" {
		return nil, errs.New("Staged commit does not contain commitID")
	}

	if response.IsErrorResponse(resp.Commit.Build.Type) {
		return &Commit{resp.Commit, nil, nil}, response.ProcessBuildError(resp.Commit.Build, "Could not process error response from stage commit")
	}

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if resp.Commit.Build.Status == raw.Planning {
		resp.Commit.Build, err = b.pollBuildPlanned(resp.Commit.CommitID.String(), params.Owner, params.Project, nil)
		if err != nil {
			return nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	bp, err := buildplan.Unmarshal(resp.Commit.Build.RawMessage)
	if err != nil {
		return nil, errs.Wrap(err, "failed to unmarshal build plan")
	}

	checkoutInfo := &buildscript.CheckoutInfo{
		Project: projectURL(params.Owner, params.Project, resp.Commit.CommitID.String()),
		AtTime:  time.Time(resp.Commit.AtTime),
	}
	stagedScript, err := buildscript.UnmarshalBuildExpression(resp.Commit.Expression, checkoutInfo)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return &Commit{resp.Commit, bp, stagedScript}, nil
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
