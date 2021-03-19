package errs

import (
	"errors"
)

// UnwrapExitCode checks if the given error is a failure of type FailExecCmdExit and
// returns the ExitCode of the process that failed with this error
func UnwrapExitCode(err error) int {
	if err == nil {
		return 0
	}

	var eerr ExitCodeable
	if errors.As(err, &eerr) {
		return eerr.ExitCode()
	}

	return 1
}
