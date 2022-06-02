package relay

// ExitCoder describes any type that can return an error code.
type ExitCoder interface {
	ExitCode() int
}

// CodedError is a simple implementaion of the ExitCoder interface.
type CodedError struct {
	Err error
	C   int
}

// Error satisfies the error interface.
func (ce *CodedError) Error() string {
	return ce.Err.Error()
}

// ExitCode satisfies the ExitCoder interface.
func (ce *CodedError) ExitCode() int {
	return ce.C
}
