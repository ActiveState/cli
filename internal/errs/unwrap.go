package errs

import (
	"errors"

	"github.com/ActiveState/cli/internal/failures"
)

func unwrapExitError(err error) (bool, int) {
	var eerr interface{ ExitCode() int }
	isExitError := errors.As(err, &eerr)
	if isExitError {
		// If exit error happened in activated shell, do not forward exit code, but return 0
		var activatedErr interface{ IsFromActivatedShell() }
		isFromActivatedShell := errors.As(err, &activatedErr)
		if isFromActivatedShell {
			return true, 0
		}

		return true, eerr.ExitCode()
	}

	return false, 0
}

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
		if errFail == nil {
			return 0
		}
		return 1
	}

	err := fail.ToError()
	isExitError = errors.As(err, &eerr)
	if isExitError {
		return eerr.ExitCode()
	}

	return 1
}
