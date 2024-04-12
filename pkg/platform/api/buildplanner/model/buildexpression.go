package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/go-openapi/strfmt"
)

func (bp *BuildPlanner) GetBuildExpression(commitID string) (*buildexpression.BuildExpression, error) {
	logging.Debug("GetBuildExpression, commitID: %s", commitID)
	resp := &bpResp.BuildExpression{}
	err := bp.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, processBuildPlannerError(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, errs.New("Commit is nil")
	}

	if bpResp.IsErrorResponse(resp.Commit.Type) {
		return nil, bpResp.ProcessCommitError(resp.Commit, "Could not get build expression from commit")
	}

	if resp.Commit.Expression == nil {
		return nil, errs.New("Commit does not contain expression")
	}

	expression, err := buildexpression.New(resp.Commit.Expression)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return expression, nil
}

func (bp *BuildPlanner) GetBuildExpressionAndTime(commitID string) (*buildexpression.BuildExpression, *strfmt.DateTime, error) {
	logging.Debug("GetBuildExpressionAndTime, commitID: %s", commitID)
	resp := &bpResp.BuildExpression{}
	err := bp.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, nil, processBuildPlannerError(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, nil, errs.New("Commit is nil")
	}

	if bpResp.IsErrorResponse(resp.Commit.Type) {
		return nil, nil, bpResp.ProcessCommitError(resp.Commit, "Could not get build expression from commit")
	}

	if resp.Commit.Expression == nil {
		return nil, nil, errs.New("Commit does not contain expression")
	}

	expression, err := buildexpression.New(resp.Commit.Expression)
	if err != nil {
		return nil, nil, errs.Wrap(err, "failed to parse build expression")
	}

	return expression, &resp.Commit.AtTime, nil
}
