package captain

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

// failureAsError converts all failures into an error
// Can't pass failures as errors and still assert them as nil, so we have to typecase.
// Blame Go for being weird.
func failureAsError(err error) error {
	err = failures.ToError(err)

	if err == nil {
		err = failures.ToError(failures.Handled())
	}

	return err
}

// UnwrapError walks through the error chain, extracts the exit code and remove non-localized errors
func UnwrapError(err error) (int, error) {
	// Ensure we are dealing with an error rather than a failure in disguise
	err = failureAsError(err)

	if err == nil {
		return 0, nil
	}

	var ee errs.Error
	stack := "not provided"
	isErrs := errors.As(err, &ee)
	if isErrs {
		stack = ee.Stack().String()
	}

	_, hasMarshaller := err.(output.Marshaller)

	// Log error if this isn't a user input error
	if !locale.IsInputError(err) {
		logging.Error("Returning error:\n%s\nCreated at:\n%s", errs.Join(err, "\n").Error(), stack)
	}

	// unwrap exit code before we remove un-localized wrapped errors from err variable
	code := unwrapExitCode(err)

	if locale.IsError(err) {
		err = locale.JoinErrors(err, "\n")
	} else if isErrs && !hasMarshaller {
		logging.Error("MUST ADDRESS: Error does not have localization: %s", errs.Join(err, "\n").Error())

		// If this wasn't built via CI then this is a dev workstation, and we should be more aggressive
		if !rtutils.BuiltViaCI {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", err.Error(), stack))
		}
	}

	if isSilentFail(err) {
		logging.Debug("Suppressing silent failure: %v", err.Error())
		err = nil
	}

	return code, err
}

func isSilentFail(errFail error) bool {
	fail, ok := errFail.(*failures.Failure)
	return ok && fail.Type.Matches(failures.FailSilent)
}

// unwrapExitCode checks if the given error is a failure of type FailExecCmdExit and
// returns the ExitCode of the process that failed with this error
func unwrapExitCode(errFail error) int {
	var eerr interface{ ExitCode() int }
	isExitError := errors.As(errFail, &eerr)
	if isExitError {
		return eerr.ExitCode()
	}

	// failure might be in the error stack
	var fail *failures.Failure
	isFailure := errors.As(errFail, &fail)
	if !isFailure {
		return 1
	}

	if !fail.Type.Matches(sscommon.FailExecCmdExit) {
		return 1
	}
	err := fail.ToError()

	isExitError = errors.As(err, &eerr)
	if isExitError {
		return eerr.ExitCode()
	}

	return 1
}
