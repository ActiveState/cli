package response

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

type BuildPlannerError struct {
	Err              error
	ValidationErrors []string
	IsTransient      bool
}

// InputError returns true as we want to treat all build planner errors as input errors
// and not report them to Rollbar. We defer the responsibility of logging these errors
// to the maintainers of the build planner.
func (e *BuildPlannerError) InputError() bool {
	return true
}

// LocaleError returns the error message to be displayed to the user.
// This function is added so that BuildPlannerErrors will be displayed
// to the user
func (e *BuildPlannerError) LocaleError() string {
	return e.Error()
}

func (e *BuildPlannerError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	// Append last five lines to error message
	offset := 0
	numLines := len(e.ValidationErrors)
	if numLines > 5 {
		offset = numLines - 5
	}

	errorLines := strings.Join(e.ValidationErrors[offset:], "\n")
	// Crop at 500 characters to reduce noisy output further
	if len(errorLines) > 500 {
		offset = len(errorLines) - 499
		errorLines = fmt.Sprintf("…%s", errorLines[offset:])
	}
	isCropped := offset > 0
	croppedMessage := ""
	if isCropped {
		croppedMessage = locale.Tl("buildplan_err_cropped_intro", "These are the last lines of the error message:")
	}

	var err error

	if croppedMessage != "" {
		err = locale.NewError("buildplan_err_cropped", "", croppedMessage, errorLines)
	} else {
		err = locale.NewError("buildplan_err", "", errorLines)
	}

	if e.IsTransient {
		err = errs.AddTips(err, locale.Tr("transient_solver_tip"))
	}

	return err.Error()
}

func (e *BuildPlannerError) Unwrap() error {
	return errors.Unwrap(e.Err)
}

func IsErrorResponse(errorType string) bool {
	return errorType == types.ErrorType ||
		errorType == types.NotFoundErrorType ||
		errorType == types.ParseErrorType ||
		errorType == types.AlreadyExistsErrorType ||
		errorType == types.NoChangeSinceLastCommitErrorType ||
		errorType == types.HeadOnBranchMovedErrorType ||
		errorType == types.ForbiddenErrorType ||
		errorType == types.RemediableSolveErrorType ||
		errorType == types.PlanningErrorType ||
		errorType == types.MergeConflictType ||
		errorType == types.FastForwardErrorType ||
		errorType == types.NoCommonBaseFoundType ||
		errorType == types.ValidationErrorType ||
		errorType == types.MergeConflictErrorType ||
		errorType == types.RevertConflictErrorType ||
		errorType == types.CommitNotInTargetHistoryErrorType ||
		errorType == types.CommitHasNoParentErrorType ||
		errorType == types.InvalidInputErrorType
}

// NotFoundError represents an error that occurred because a resource was not found.
type NotFoundError struct {
	Type                  string `json:"type"`
	Resource              string `json:"resource"`
	MayNeedAuthentication bool   `json:"mayNeedAuthentication"`
}

// ParseError is an error that occurred while parsing the build expression.
type ParseError struct {
	Path string `json:"path"`
}

type ForbiddenError struct {
	Operation string `json:"operation"`
}

// Error contains an error message.
type Error struct {
	Message string `json:"message"`
}

type TargetNotFoundError struct {
	Message         string
	RequestedTarget string
	PossibleTargets []string
}

func (e *TargetNotFoundError) Error() string {
	return e.Message
}

func (e *TargetNotFoundError) InputError() bool {
	return true
}
