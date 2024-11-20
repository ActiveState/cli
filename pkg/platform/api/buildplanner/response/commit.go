package response

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

func ProcessBuildError(build *BuildResponse, fallbackMessage string) error {
	logging.Debug("ProcessBuildError: build.Type=%s", build.Type)
	if build.Type == types.PlanningErrorType {
		return processPlanningError(build.Message, build.SubErrors)
	} else if build.Error == nil {
		return errs.New(fallbackMessage)
	}

	return locale.NewInputError("err_buildplanner_build", "Encountered error processing build response")
}

func processPlanningError(message string, subErrors []*BuildExprError) error {
	var errs []string
	var isTransient bool

	if message != "" {
		errs = append(errs, message)
	}

	for _, se := range subErrors {
		if se.Type == types.TargetNotFoundErrorType {
			return &TargetNotFoundError{
				Message:         se.Message,
				RequestedTarget: se.RequestedTarget,
				PossibleTargets: se.PossibleTargets,
			}
		}

		if se.Type != types.RemediableSolveErrorType && se.Type != types.GenericSolveErrorType {
			continue
		}

		if se.Message != "" {
			errs = append(errs, se.Message)
			isTransient = se.IsTransient
		}

		for _, ve := range se.ValidationErrors {
			if ve.Error != "" {
				errs = append(errs, ve.Error)
			}
		}
	}
	return &BuildPlannerError{
		ValidationErrors: errs,
		IsTransient:      isTransient,
	}
}

func ProcessProjectError(project *ProjectResponse, fallbackMessage string) error {
	if project.Type == types.NotFoundErrorType {
		return errs.AddTips(
			locale.NewInputError("err_buildplanner_project_not_found", "Unable to find project. Received message: {{.V0}}", project.Message),
			locale.T("tip_private_project_auth"),
		)
	}

	return errs.New(fallbackMessage)
}

// Commit contains the build and any errors.
type Commit struct {
	Type       string          `json:"__typename"`
	AtTime     strfmt.DateTime `json:"atTime"`
	Expression json.RawMessage `json:"expr"`
	CommitID   strfmt.UUID     `json:"commitId"`
	ParentID   strfmt.UUID     `json:"parentId"`
	Build      *BuildResponse  `json:"build"`
	*Error
	*ErrorWithSubErrors
}
