package buildplanner

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

type ImpactReportParams struct {
	Owner          string
	Project        string
	BeforeCommitId strfmt.UUID
	AfterExpr      []byte
}

func (b *BuildPlanner) ImpactReport(params *ImpactReportParams) (*response.ImpactReportResult, error) {
	request := request.ImpactReport(params.Owner, params.Project, params.BeforeCommitId, params.AfterExpr)
	resp := &response.ImpactReportResponse{}
	err := b.client.Run(request, resp)
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
