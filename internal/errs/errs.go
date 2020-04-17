package errs

import (
	"fmt"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
)

type Error struct {
	error
	wrapped error
	stack   *stacktrace.Stacktrace
}

func (e *Error) Unwrap() error {
	return e.wrapped
}

func (e *Error) Stack() *stacktrace.Stacktrace {
	return e.stack
}

type UserInputError struct {
	error
}

func (e *UserInputError) Unwrap() error {
	return e.error
}

var UserInputErr = &UserInputError{}

func newError(err error, wrapTarget error) error {
	return &Error{
		err,
		wrapTarget,
		stacktrace.GetWithSkip([]string{rtutils.CurrentFile()}),
	}
}

// ToError ensures the given error is wrapped in errs.Error
func ToError(err error) error {
	if err == nil {
		return nil
	}
	if ee, ok := err.(*Error); ok {
		return ee
	}
	return newError(err, nil)
}

// New creates a new error, similar to errors.New
func New(message string, args ...interface{}) error {
	return newError(fmt.Errorf(message, args...), nil)
}

// Wrap will wrap one error around another, allowing it to unwrap to the wrapTarget
func Wrap(err error, wrapTarget error) error {
	// Just amend the existing error if we already have an errs.Error type and the wrapped is nil
	ee := ToError(err).(*Error)
	if ee.wrapped == nil {
		ee.wrapped = wrapTarget
		return ee
	}
	
	return newError(err, wrapTarget)
}
