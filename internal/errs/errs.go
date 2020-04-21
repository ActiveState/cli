package errs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
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
	msg     string
	wrapped error
	stack   *stacktrace.Stacktrace
}

// Error returns the error message
func (e *WrappedErr) Error() string {
	return e.msg
}

// Unwrap returns the parent error, if one exists
func (e *WrappedErr) Unwrap() error {
	return e.wrapped
}

// Stack returns the stacktrace for where this error was created
func (e *WrappedErr) Stack() *stacktrace.Stacktrace {
	return e.stack
}

func newError(err string, wrapTarget error) error {
	return &WrappedErr{
		err,
		wrapTarget,
		stacktrace.GetWithSkip([]string{rtutils.CurrentFile()}),
	}
}

// New creates a new error, similar to errors.New
func New(message string, args ...interface{}) error {
	return newError(fmt.Sprintf(message, args...), nil)
}

// Wrap creates a new error that wraps the given error
func Wrap(wrapTarget error, message string, args ...interface{}) error {
	return newError(fmt.Sprintf(message, args...), wrapTarget)
}

// Join all error messages in the Unwrap stack
func Join(err error, sep string) error {
	var message []string
	for err != nil {
		message = append(message, err.Error())
		err = errors.Unwrap(err)
	}
	return Wrap(err, strings.Join(message, sep))
}
