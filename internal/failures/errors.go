package failures

type ExitError struct {
	Err    error
	Silent bool
}

func (e *ExitError) IsSilent() bool {
	return e.Silent
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown exit error"
}
