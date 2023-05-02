package errs

import (
	"errors"
)

// ParseExitCode checks if the given error is a failure of type ExitCodeable and
// returns the ExitCode of the process that failed with this error
func ParseExitCode(err error) int {
	if err == nil {
		return 0
	}

	var eerr ExitCodeable
	if errors.As(err, &eerr) {
		return eerr.ExitCode()
	}

	return 1
}
