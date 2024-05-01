package locale

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
)

var _ ErrorLocalizer = &LocalizedError{}

// LocalizedError is an error that has the concept of user facing (localized) errors as well as whether an error is due
// to user input or not
type LocalizedError struct {
	wrapped     error
	tips        []string
	localized   string
	stack       *stacktrace.Stacktrace
	inputErr    bool
	externalErr bool
}

// Error is the error message
func (e *LocalizedError) Error() string {
	return e.localized
}

// LocaleError is the user facing error message, it's the same as Error() but identifies it as being user facing
func (e *LocalizedError) LocaleError() string {
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

func (e *LocalizedError) ErrorTips() []string {
	return e.tips
}

func (e *LocalizedError) ExternalError() bool {
	return e.externalErr
}

func (e *LocalizedError) AddTips(tips ...string) {
	e.tips = append(e.tips, tips...)
}

// ErrorLocalizer represents a localized error
type ErrorLocalizer interface {
	error
	LocaleError() string
}

type AsError interface {
	As(interface{}) bool
}

// ErrorInput represents a user input error
type ErrorInput interface {
	InputError() bool
}

// NewError creates a new error, it does a locale.Tl lookup of the given id, if the lookup fails it will use the
// locale string instead
func NewError(id string, args ...string) *LocalizedError {
	return WrapError(nil, id, args...)
}

// WrapError creates a new error that wraps the given error, it does a locale.Tt lookup of the given id, if the lookup
// fails it will use the locale string instead
func WrapError(err error, id string, args ...string) *LocalizedError {
	locale := id
	if len(args) > 0 {
		locale, args = args[0], args[1:]
	}

	l := &LocalizedError{}
	translation := Tl(id, locale, args...)
	l.wrapped = err
	l.tips = []string{}
	l.localized = translation
	l.stack = stacktrace.GetWithSkip([]string{rtutils.CurrentFile()})

	return l
}

// NewInputError is like NewError but marks it as an input error
func NewInputError(id string, args ...string) *LocalizedError {
	return WrapInputError(nil, id, args...)
}

// WrapInputError is like WrapError but marks it as an input error
func WrapInputError(err error, id string, args ...string) *LocalizedError {
	locale := id
	if len(args) > 0 {
		locale, args = args[0], args[1:]
	}
	if locale == "" {
		locale = id
	}

	l := &LocalizedError{}
	translation := Tl(id, locale, args...)
	l.inputErr = true
	l.externalErr = true
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

// HasError checks the error chain for an ErrorLocalizer
func HasError(err error) bool {
	var el ErrorLocalizer
	return errors.As(err, &el)
}

// IsInputError checks if the given error contains a InputError anywhere in the unwrap stack
func IsInputError(err error) bool {
	if err == nil {
		return false
	}
	for _, err := range errs.Unpack(err) {
		errInput, ok := err.(ErrorInput)
		if ok && errInput.InputError() {
			return true
		}
	}
	return false
}

// IsInputErrorNonRecursive checks if the given error is an InputError
func IsInputErrorNonRecursive(err error) bool {
	if err == nil {
		return false
	}
	errInput, ok := err.(ErrorInput)
	if ok && errInput.InputError() {
		return true
	}
	return false
}

// JoinedErrorMessage joins all error messages in the Unwrap stack that are localized
func JoinedErrorMessage(err error) string {
	var message []string
	for _, err := range UnpackError(err) {
		if lerr, isLocaleError := err.(ErrorLocalizer); isLocaleError {
			message = append(message, lerr.LocaleError())
		}
	}
	if len(message) == 0 {
		if !condition.BuiltViaCI() {
			panic(fmt.Sprintf("Errors must be localized! Please localize: %s, called at: %s\n", errs.JoinMessage(err), stacktrace.Get()))
		}
		multilog.Error("MUST ADDRESS: Error does not have localization: %s", errs.JoinMessage(err))
		return err.Error()
	}
	return strings.Join(message, ": ")
}

func ErrorMessage(err error) string {
	if errr, ok := err.(ErrorLocalizer); ok {
		return errr.LocaleError()
	}
	return err.Error()
}

// UnpackError recursively unpacks the given error and returns all localized errors
func UnpackError(err error) []error {
	var errors []error
	for _, err := range errs.Unpack(err) {
		lerr, isLocaleError := err.(ErrorLocalizer)
		if isLocaleError {
			errors = append(errors, lerr)
		}
	}

	return errors
}
