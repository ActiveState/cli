package errs

type ExitCodeable interface {
	ExitCode() int
}

type ExitCode struct {
	code       int
	wrappedErr error
	silent     bool
}

func WrapExitCode(err error, code int, silent bool) error {
	return &ExitCode{code, err, silent}
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

func (e *ExitCode) IsSilent() bool {
	return e.silent
}
