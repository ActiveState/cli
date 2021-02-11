package errs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
)

// WrapperError enforces errors that include a stacktrace
type Errorable interface {
	Unwrap() error
	Stack() *stacktrace.Stacktrace
}

type ErrorTips interface {
	AddTips(...string)
	ErrorTips() []string
}

// WrapperError is what we use for errors created from this package, this does not mean every error returned from this
// package is wrapping something, it simply has the plumbing to.
type WrapperError struct {
	message string
	tips    []string
	wrapped error
	stack   *stacktrace.Stacktrace
}

func (e *WrapperError) Error() string {
	return e.message
}

func (e *WrapperError) ErrorTips() []string {
	return e.tips
}

func (e *WrapperError) AddTips(tips ...string) {
	e.tips = append(e.tips, tips...)
}

// Unwrap returns the parent error, if one exists
func (e *WrapperError) Unwrap() error {
	return e.wrapped
}

// Stack returns the stacktrace for where this error was created
func (e *WrapperError) Stack() *stacktrace.Stacktrace {
	return e.stack
}

func newError(message string, wrapTarget error) *WrapperError {
	return &WrapperError{
		message,
		[]string{},
		wrapTarget,
		stacktrace.GetWithSkip([]string{rtutils.CurrentFile()}),
	}
}

// New creates a new error, similar to errors.New
func New(message string, args ...interface{}) *WrapperError {
	return newError(fmt.Sprintf(message, args...), nil)
}

// Wrap creates a new error that wraps the given error
func Wrap(wrapTarget error, message string, args ...interface{}) *WrapperError {
	return newError(fmt.Sprintf(message, args...), wrapTarget)
}

// Join all error messages in the Unwrap stack
func Join(err error, sep string) *WrapperError {
	var message []string
	for err != nil {
		message = append(message, err.Error())
		err = errors.Unwrap(err)
	}
	return Wrap(err, strings.Join(message, sep))
}

func AddTips(err error, tips ...string) error {
	if _, ok := err.(ErrorTips); !ok {
		err = newError("wrapped error to add tips", err)
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

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// Matches is an analog for errors.As that just checks whether err matches the given type, so you can do:
// errs.Matches(err, &ErrStruct{})
// Without having to first assign it to a variable
// This is useful if you ONLY care about the bool return value and not about setting the variable
func Matches(err error, target interface{}) bool {
	if target == nil {
		panic("errors: target cannot be nil")
	}

	val := reflect.ValueOf(target)
	targetType := val.Type()
	if targetType.Kind() != reflect.Interface && !targetType.Implements(errorType) {
		panic("errors: *target must be interface or implement error")
	}
	for err != nil {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			return true
		}
		if x, ok := err.(interface{ As(interface{}) bool }); ok && x.As(&target) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
