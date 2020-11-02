package errs

import (
	"errors"

	"github.com/ActiveState/cli/internal/failures"
)

// UnwrapExitCode checks if the given error is a failure of type FailExecCmdExit and
// returns the ExitCode of the process that failed with this error
func UnwrapExitCode(errFail error) int {
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

	if !fail.Type.Matches(failures.FailExecCmdExit) {
		return 1
	}
	err := fail.ToError()

	isExitError = errors.As(err, &eerr)
	if isExitError {
		return eerr.ExitCode()
	}

	return 1
}
