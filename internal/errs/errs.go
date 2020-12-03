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

type ErrorTips interface {
	AddTips(...string)
	ErrorTips() []string
}

// WrappedErr is what we use for errors created from this package, this does not mean every error returned from this
// package is wrapping something, it simply has the plumbing to.
type WrappedErr struct {
	error
	tips    []string
	wrapped error
	stack   *stacktrace.Stacktrace
}

func (e *WrappedErr) Error() string {
	if e.error != nil {
		return e.error.Error()
	}
	if e.wrapped != nil {
		return e.wrapped.Error()
	}
	return "incorrectly wrapped error"
}

func (e *WrappedErr) ErrorTips() []string {
	return e.tips
}

func (e *WrappedErr) AddTips(tips ...string) {
	e.tips = append(e.tips, tips...)
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
		[]string{},
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
	for err != nil {
		message = append(message, err.Error())
		err = errors.Unwrap(err)
	}
	return Wrap(err, strings.Join(message, sep))
}

type ErrorWithTips struct {
	error
	tips []string
}

func (e *ErrorWithTips) ErrorTips() []string {
	return e.tips
}

func AddTips(err error, tips ...string) error {
	if _, ok := err.(ErrorTips); !ok {
		err = newError(nil, err)
	}
	err.(ErrorTips).AddTips(tips...)
	return err
}

// InnerError unwraps wrapped error messages
func InnerError(err error) error {
	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		return InnerError(unwrapped)
	}
	return err
}

// Matches checks if err matches the given target errors type
func Matches(err error, target error) bool {
	for err != nil {
		if errors.Is(err, target) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
