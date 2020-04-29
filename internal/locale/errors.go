package locale

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
)

// LocalizedError is an error that has the concept of user facing (localized) errors as well as whether an error is due
// to user input or not
type LocalizedError struct {
	wrapped   error
	localized string
	stack     *stacktrace.Stacktrace
	inputErr  bool
}

// Error is the error message
func (e *LocalizedError) Error() string {
	return e.localized
}

// UserError is the user facing error message, it's the same as Error() but identifies it as being user facing
func (e *LocalizedError) UserError() string {
	return e.localized
}

// Stack is the stacktrace leading up to where this error was triggered
func (e *LocalizedError) Stack() *stacktrace.Stacktrace {
	return e.stack
}

// Unwrap returns the parent error, if applicable
func (e *LocalizedError) Unwrap() error {
	return e.wrapped
}

// InputError returns whether this is an error due to user input
func (e *LocalizedError) InputError() bool {
	return e.inputErr
}

// ErrorLocalizer represents a localized error
type ErrorLocalizer interface {
	UserError() string
}

// ErrorInput represents a user input error
type ErrorInput interface {
	InputError() bool
}

// NewError creates a new error, it does a locale.Tt lookup of the given id, if the lookup fails it will use the
// locale string instead
func NewError(id, locale string, args ...string) error {
	return WrapError(nil, id, locale, args...)
}

// WrapError creates a new error that wraps the given error, it does a locale.Tt lookup of the given id, if the lookup
// fails it will use the locale string instead
func WrapError(err error, id, locale string, args ...string) error {
	l := &LocalizedError{}
	translation := Tl(id, locale, args...)
	l.wrapped = err
	l.localized = translation
	l.stack = stacktrace.GetWithSkip([]string{rtutils.CurrentFile()})
	return l
}

// NewInputError is like NewError but marks it as an input error
func NewInputError(id, locale string, args ...string) error {
	return WrapInputError(nil, id, locale, args...)
}

// WrapInputError is like WrapError but marks it as an input error
func WrapInputError(err error, id, locale string, args ...string) error {
	l := &LocalizedError{}
	translation := Tl(id, locale, args...)
	l.inputErr = true
	l.wrapped = err
	l.localized = translation
	l.stack = stacktrace.GetWithSkip([]string{rtutils.CurrentFile()})
	return l
}

// IsError checks if the given error is an ErrorLocalizer
func IsError(err error) bool {
	_, ok := err.(ErrorLocalizer)
	return ok
}

// IsInputError checks if the given error contains a InputError anywhere in the unwrap stack
func IsInputError(err error) bool {
	if err == nil {
		return false
	}
	var errInput ErrorInput = &LocalizedError{}
	for err != nil && errors.As(err, &errInput) {
		if errors.As(err, &errInput) && errInput.InputError() {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

type failure interface {
	IsFailure()
}

// JoinErrors joins all error messages in the Unwrap stack that are localized
func JoinErrors(err error, sep string) error {
	var message []string
	for err != nil {
		if errr, ok := err.(ErrorLocalizer); ok {
			message = append(message, errr.UserError())
		}
		if _, ok := err.(failure); ok { // For now we include failures, but this is a deprecated mechanic
			message = append(message, err.Error())
		}
		err = errors.Unwrap(err)
	}
	return WrapError(err, "", strings.Join(message, sep))
}
