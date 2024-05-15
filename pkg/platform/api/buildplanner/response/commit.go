package response

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

type ProjectCommitResponse struct {
	Project *ProjectResponse `json:"project"`
}

// PostProcess must satisfy gqlclient.PostProcessor interface
func (c *ProjectCommitResponse) PostProcess() error {
	if c.Project == nil {
		return errs.New("Project is nil")
	}

	if IsErrorResponse(c.Project.Type) {
		return ProcessProjectError(c.Project, "Could not get build from project response")
	}

	if c.Project.Commit == nil {
		return errs.New("Commit is nil")
	}

	if IsErrorResponse(c.Project.Type) {
		return ProcessProjectError(c.Project, "Could not get build from project response")
	}

	if c.Project.Commit == nil {
		return errs.New("Commit is nil")
	}

	if IsErrorResponse(c.Project.Commit.Type) {
		return ProcessCommitError(c.Project.Commit, "Could not get build from commit from project response")
	}

	if c.Project.Commit.Build == nil {
		return errs.New("Commit does not contain build")
	}

	if IsErrorResponse(c.Project.Commit.Build.Type) {
		return ProcessBuildError(c.Project.Commit.Build, "Could not get build from project commit response")
	}

	return nil
}

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
			locale.NewInputError("err_buildplanner_project_not_found", "Unable to find project, received message: {{.V0}}", project.Message),
			locale.T("tip_private_project_auth"),
		)
	}

	return errs.New(fallbackMessage)
}

// PlanningError represents an error that occurred during planning.
type PlanningError struct {
	SubErrors []*BuildExprError `json:"subErrors"`
}

// BuildExprError represents a location in the build script where an error occurred.
type BuildExprError struct {
	Type             string                        `json:"__typename"`
	BuildExprPath    string                        `json:"buildExprPath"`
	Message          string                        `json:"message"`
	IsTransient      bool                          `json:"isTransient"`
	ValidationErrors []*SolverErrorValidationError `json:"validationErrors"`
	*TargetNotFound
	*RemediableSolveError
}

type TargetNotFound struct {
	Type            string   `json:"__typename"`
	RequestedTarget string   `json:"requestedTarget"`
	PossibleTargets []string `json:"possibleTargets"`
}

type ValidationError struct {
	SubErrors []*BuildExprError `json:"subErrors"`
}

// Commit contains the build and any errors.
type Commit struct {
	Type       string          `json:"__typename"`
	AtTime     strfmt.DateTime `json:"atTime"`
	Expression json.RawMessage `json:"expr"`
	CommitID   strfmt.UUID     `json:"commitId"`
	Build      *BuildResponse  `json:"build"`
	*Error
	*ParseError
	*ValidationError
	*ForbiddenError
	*HeadOnBranchMovedError
	*NoChangeSinceLastCommitError
}
