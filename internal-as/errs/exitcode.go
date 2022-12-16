package errs

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
