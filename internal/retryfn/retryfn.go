package retryfn

import (
	"errors"
	"fmt"
)

// Control groups directives for retry function control.
type Control int

// Control constants are a set of potential control directives.
const (
	Unset Control = iota
	Unknown
	Halt
)

// ControlError represents an inspectable error used to control retry function
// behavior.
type ControlError struct {
	Cause error
	Type  Control
	tries int
}

// Error implements the error interface.
func (e *ControlError) Error() string {
	return fmt.Sprintf("retryfn (control type: %s): %v", e.Type.String(), e.Cause)
}

// Unwrap allows the causing error to be inspected.
func (e *ControlError) Unwrap() error {
	return e.Cause
}

// String implements the fmt.Stringer interface.
func (c Control) String() string {
	switch c {
	case Unset:
		return "unset"
	case Halt:
		return "halt"
	default:
		return "unknown"
	}
}

// RetryFn manages a retryable function.
type RetryFn struct {
	tries int
	fn    func() error
	calls int
}

// New returns a new instance of RetryFn.
func New(tries int, fn func() error) *RetryFn {
	return &RetryFn{
		tries: tries,
		fn:    fn,
	}
}

// Run calls the retryable function.
func (rf *RetryFn) Run() error {
	if rf.fn == nil {
		return nil
	}

	var err error
	for i := rf.tries; i > 0; i-- {
		rf.calls++

		err = rf.fn()
		if err == nil {
			continue
		}

		cerr := &ControlError{}
		if !errors.As(err, &cerr) {
			continue
		}

		switch cerr.Type {
		case Unset, Unknown:
			err = cerr.Cause
			continue

		case Halt:
			return cerr.Cause
		}
	}

	return err
}

// Calls returns the amount of times the managed function has been retried.
func (rf *RetryFn) Calls() int {
	return rf.calls
}
