package buildplanner

import (
	"encoding/json"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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

const numRetries = 10
const retryInterval = 500 * time.Millisecond

func (b *BuildPlanner) ImpactReport(params *ImpactReportParams) (*response.ImpactReportResult, error) {
	beforeExpr, err := json.Marshal(params.Before)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to marshal old buildexpression")
	}

	afterExpr, err := json.Marshal(params.After)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to marshal buildexpression")
	}

	var resp *response.ImpactReportResponse
	for i := 0; i < numRetries; i++ {
		resp, err = run(b.client, params.Owner, params.Project, beforeExpr, afterExpr)
		if err == nil || !response.IsImpactReportBuildPlanningError(err) {
			break
		}
		logging.Debug("Impact report response was that the buildplanner was still planning; trying again")
		time.Sleep(retryInterval * time.Duration(i+1))
	}
	if err != nil {
		return nil, errs.Wrap(err, "failed to get impact report")
	}

	return resp.ImpactReportResult, nil
}

func run(client *client, owner, project string, beforeExpr, afterExpr []byte) (*response.ImpactReportResponse, error) {
	request := request.ImpactReport(owner, project, beforeExpr, afterExpr)
	resp := &response.ImpactReportResponse{}
	err := client.Run(request, resp)
	if err != nil {
		return nil, processBuildPlannerError(err, "failed to get impact report")
	}

	if resp.ImpactReportResult == nil {
		return nil, errs.New("ImpactReport is nil")
	}

	if response.IsErrorResponse(resp.ImpactReportResult.Type) {
		return nil, response.ProcessImpactReportError(resp.ImpactReportResult, "Could not get impact report")
	}

	return resp, nil
}
