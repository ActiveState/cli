package buildplanner

import (
	"github.com/ActiveState/cli/internal/errs"

	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func (bp *BuildPlanner) Evaluate(org, project string, script *buildscript.BuildScript) error {
	expression, err := script.MarshalBuildExpression()
	if err != nil {
		return errs.Wrap(err, "Failed to marshal build expression")
	}

	request := request.Evaluate(org, project, expression, script.AtTime(), script.Dynamic(), "")
	resp := &response.BuildResponse{}
	err = bp.client.Run(request, resp)
	if err != nil {
		return processBuildPlannerError(err, "Failed to evaluate build expression")
	}

	return nil
}
