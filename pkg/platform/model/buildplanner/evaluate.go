package buildplanner

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func (bp *BuildPlanner) Evaluate(org, project string, script *buildscript.BuildScript) error {
	expression, err := script.MarshalBuildExpression()
	if err != nil {
		return errs.Wrap(err, "Failed to marshal build expression")
	}

	// Evaluate is not done until the build plan is ready
	var sessionId strfmt.UUID
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			resp := &response.EvaluateResponse{}
			req := request.Evaluate(org, project, expression, sessionId, script.AtTime(), script.Dynamic())
			err = bp.client.Run(req, resp)
			if err != nil {
				return processBuildPlannerError(err, "Failed to evaluate build expression")
			}
			sessionId = resp.SessionID
			if resp.Status != raw.Planning && resp.Status != raw.Started {
				return nil
			}
		case <-time.After(pollTimeout):
			return locale.NewError("err_buildplanner_timeout", "Timed out waiting for evaluation of build plan")
		}
	}
}
