package buildplanner

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

type ImpactReportParams struct {
	Owner   string
	Project string
	Before  *buildscript.BuildScript
	After   *buildscript.BuildScript
}

func (b *BuildPlanner) ImpactReport(params *ImpactReportParams) (*response.ImpactReportResult, error) {
	beforeExpr, err := json.Marshal(params.Before)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to marshal old buildexpression")
	}

	afterExpr, err := json.Marshal(params.After)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to marshal buildexpression")
	}

	request := request.ImpactReport(params.Owner, params.Project, beforeExpr, afterExpr)
	resp := &response.ImpactReportResponse{}
	err = b.client.Run(request, resp)
	if err != nil {
		return nil, processBuildPlannerError(err, "failed to get impact report")
	}

	if resp.ImpactReportResult == nil {
		return nil, errs.New("ImpactReport is nil")
	}

	if response.IsErrorResponse(resp.ImpactReportResult.Type) {
		return nil, response.ProcessImpactReportError(resp.ImpactReportResult, "Could not get impact report")
	}

	return resp.ImpactReportResult, nil
}
