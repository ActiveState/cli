package errs

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/rtutils"
)

// Error enforces errors that include a stacktrace
type Error interface {
	Unwrap() error
	Stack() *stacktrace.Stacktrace
}

// WrappedErr is what we use for errors created from this package, this does not mean every error returned from this
// package is wrapping something, it simply has the plumbing to.
type WrappedErr struct {
	error
	wrapped error
	stack   *stacktrace.Stacktrace
}

// Unwrap returns the parent error, if one exists
func (e *WrappedErr) Unwrap() error {
	return e.wrapped
}

// Stack returns the stacktrace for where this error was created
func (e *WrappedErr) Stack() *stacktrace.Stacktrace {
	return e.stack
}

func newError(err error, wrapTarget error) error {
	return &WrappedErr{
		err,
		wrapTarget,
		stacktrace.GetWithSkip([]string{rtutils.CurrentFile()}),
	}
}

// New creates a new error, similar to errors.New
func New(message string, args ...interface{}) error {
	return newError(errors.New(fmt.Sprintf(message, args...)), nil)
}

// Wrap creates a new error that wraps the given error
func Wrap(wrapTarget error, message string, args ...interface{}) error {
	return newError(errors.New(fmt.Sprintf(message, args...)), wrapTarget)
}

// WrapErrors wraps one error in another
func WrapErrors(wrapTarget error, wrapper error) error {
	return newError(wrapper, wrapTarget)
}

// Join all error messages in the Unwrap stack
func Join(err error, sep string) error {
	var message []string
	for !errIsNil(err) {
		message = append(message, err.Error())
		err = errors.Unwrap(err)
	}
	return Wrap(err, strings.Join(message, sep))
}

// errIsNil is a dirty little helper function that helps surface fail=nil type issues, to be removed once we get rid of failures
func errIsNil(err error) bool {
	if fail, ok := err.(*failures.Failure); ok && fail == nil && err != nil {
		logging.Error("MUST FIX: nil failure is being passed as non-nil error, os.Args: %v", os.Args)
		if !rtutils.BuiltViaCI {
			// Ensure we don't miss this while testing locally
			print.Error("MUST FIX: nil failure is being passed as non-nil error")
		}
		return true
	}
	return err == nil
}

// InnerError unwraps wrapped error messages
func InnerError(err error) error {
	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		return InnerError(unwrapped)
	}
	return err
}
