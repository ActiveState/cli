package errs

import "errors"

type ExitCodeable interface {
	ExitCode() int
}

type ExitCode struct {
	code       int
	wrappedErr error
}

func WrapExitCode(err error, code int) error {
	return &ExitCode{code, err}
}

func (e *ExitCode) Error() string {
	return "ExitCode"
}

func (e *ExitCode) Unwrap() error {
	return e.wrappedErr
}

func (e *ExitCode) ExitCode() int {
	return e.code
}

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
