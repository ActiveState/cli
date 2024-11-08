package buildplanner

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, params.Description, script.AtTime(), expression)
	resp := &response.Commit{}
	if err := b.client.Run(request, resp); err != nil {
		return nil, processBuildPlannerError(err, "failed to stage commit")
	}

	if resp == nil {
		return nil, errs.New("Staged commit is nil")
	}

	if response.IsErrorResponse(resp.Type) {
		return nil, response.ProcessCommitError(resp, "Could not process error response from stage commit")
	}

	if resp.CommitID == "" {
		return nil, errs.New("Staged commit does not contain commitID")
	}

	if response.IsErrorResponse(resp.Build.Type) {
		return &Commit{resp, nil, nil}, response.ProcessBuildError(resp.Build, "Could not process error response from stage commit")
	}

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if resp.Build.Status == raw.Planning {
		resp.Build, err = b.pollBuildPlanned(resp.CommitID.String(), params.Owner, params.Project, nil)
		if err != nil {
			return nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	bp, err := buildplan.Unmarshal(resp.Build.RawMessage)
	if err != nil {
		return nil, errs.Wrap(err, "failed to unmarshal build plan")
	}

	stagedScript := buildscript.New()
	stagedScript.SetAtTime(time.Time(resp.AtTime), false)
	if err := stagedScript.UnmarshalBuildExpression(resp.Expression); err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return &Commit{resp, bp, stagedScript}, nil
}

func (b *BuildPlanner) RevertCommit(organization, project, parentCommitID, commitID string) (strfmt.UUID, error) {
	logging.Debug("RevertCommit, organization: %s, project: %s, commitID: %s", organization, project, commitID)
	resp := &response.RevertedCommit{}
	err := b.client.Run(request.RevertCommit(organization, project, parentCommitID, commitID), resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to revert commit")
	}

	if resp == nil {
		return "", errs.New("Revert commit response is nil")
	}

	if response.IsErrorResponse(resp.Type) {
		return "", response.ProcessRevertCommitError(resp, "Could not revert commit")
	}

	if resp.Commit == nil {
		return "", errs.New("Revert commit's commit is nil'")
	}

	if response.IsErrorResponse(resp.Commit.Type) {
		return "", response.ProcessCommitError(resp.Commit, "Could not process error response from revert commit")
	}

	if resp.Commit.CommitID == "" {
		return "", errs.New("Commit does not contain commitID")
	}

	return resp.Commit.CommitID, nil
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
	resp := &response.MergedCommit{}
	err := b.client.Run(request, resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to merge commit")
	}

	if resp == nil {
		return "", errs.New("MergedCommit is nil")
	}

	if response.IsErrorResponse(resp.Type) {
		return "", response.ProcessMergedCommitError(resp, "Could not merge commit")
	}

	if resp.Commit == nil {
		return "", errs.New("Merge commit's commit is nil'")
	}

	if response.IsErrorResponse(resp.Commit.Type) {
		return "", response.ProcessCommitError(resp.Commit, "Could not process error response from merge commit")
	}

	return resp.Commit.CommitID, nil
}
