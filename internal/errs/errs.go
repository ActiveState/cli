package errs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils"
	"gopkg.in/yaml.v3"
)

const TipMessage = "wrapped tips"

type AsError interface {
	As(interface{}) bool
}

// WrapperError enforces errors that include a stacktrace
type Errorable interface {
	Unwrap() error
	Stack() *stacktrace.Stacktrace
}

type ErrorTips interface {
	error
	AddTips(...string)
	ErrorTips() []string
}

// TransientError represents an error that is transient, meaning it does not itself represent a failure, but rather it
// facilitates a mechanic meant to get to the actual error (eg. by wrapping or packing underlying errors).
// Do NOT satisfy this interface for errors whose type you want to assert.
type TransientError interface {
	IsTransient()
}

// PackedErrors represents a collection of errors that aren't necessarily related to each other
// note that rtutils replicates this functionality to avoid import cycles
type PackedErrors struct {
	errors []error
}

func (e *PackedErrors) IsTransient() {}

func (e *PackedErrors) Error() string {
	return "packed multiple errors"
}

func (e *PackedErrors) Unwrap() []error {
	return e.errors
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
	msg := fmt.Sprintf(message, args...)
	return newError(msg, nil)
}

// Wrap creates a new error that wraps the given error
func Wrap(wrapTarget error, message string, args ...interface{}) *WrapperError {
	msg := fmt.Sprintf(message, args...)
	return newError(msg, wrapTarget)
}

// Pack creates a new error that packs the given errors together, allowing for multiple errors to be returned
func Pack(err error, errs ...error) error {
	return &PackedErrors{append([]error{err}, errs...)}
}

// encodeErrorForJoin will recursively encode an error into a format that can be marshalled in a way that is easily
// humanly readable.
// In a nutshell the logic is:
// - If the error is nil, return nil
// - If the error is wrapped other errors, return it as a map with the key being the error and the value being the wrapped error(s)
// - If the error is packing other errors, return them as a list of errors
func encodeErrorForJoin(err error) interface{} {
	if err == nil {
		return nil
	}

	// If the error is a wrapper, unwrap it and encode the wrapped error
	if u, ok := err.(unwrapNext); ok {
		subErr := u.Unwrap()
		if subErr == nil {
			return err.Error()
		}
		return map[string]interface{}{err.Error(): encodeErrorForJoin(subErr)}
	}

	// If the error is a packer, encode the packed errors as a list
	if u, ok := err.(unwrapPacked); ok {
		var result []interface{}
		// Don't encode errors that are transient as the real errors are kept underneath
		if _, isTransient := err.(TransientError); !isTransient {
			result = append(result, err.Error())
		}
		errs := u.Unwrap()
		for _, nextErr := range errs {
			result = append(result, encodeErrorForJoin(nextErr))
		}
		if len(result) == 1 {
			return result[0]
		}
		return result
	}

	return err.Error()
}

func JoinMessage(err error) string {
	v, err := yaml.Marshal(encodeErrorForJoin(err))
	if err != nil {
		// This shouldn't happen since we know exactly what we are marshalling
		return fmt.Sprintf("failed to marshal error: %s", err)
	}
	return strings.TrimSpace(string(v))
}

func AddTips(err error, tips ...string) error {
	var errTips ErrorTips
	// MultiError uses a custom type to wrap multiple errors, so the type casting above won't work.
	// Instead it satisfied `errors.As()`, but here we want to specifically check the current error and not any wrapped errors.
	if asError, ok := err.(AsError); ok {
		asError.As(&errTips)
	}
	if _, ok := err.(ErrorTips); ok {
		errTips = err.(ErrorTips)
	}
	if errTips == nil {
		// use original error message with identifier in case this bubbles all the way up
		// this helps us identify it on rollbar without affecting the UX too much
		errTips = newError(TipMessage, err)
		err = errTips
	}
	errTips.AddTips(tips...)
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
	errs := Unpack(err)
	for _, err := range errs {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			return true
		}
		if x, ok := err.(interface{ As(interface{}) bool }); ok && x.As(&target) {
			return true
		}
	}
	return false
}

func IsAny(err error, errs ...error) bool {
	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

type unwrapNext interface {
	Unwrap() error
}

type unwrapPacked interface {
	Unwrap() []error
}

// Unpack will recursively unpack an error into a list of errors, which is useful if you need to iterate over all errors.
// This is similar to errors.Unwrap, but will also "unwrap" errors that are packed together, which errors.Unwrap does not.
func Unpack(err error) []error {
	result := []error{}

	// add is a little helper function to add errors to the result, skipping any transient errors
	add := func(errors ...error) {
		for _, err := range errors {
			if _, isTransient := err.(TransientError); isTransient {
				continue
			}
			result = append(result, err)
		}
	}

	// recursively unpack the error
	for err != nil {
		add(err)
		if u, ok := err.(unwrapNext); ok {
			// The error implements `Unwrap() error`, so simply unwrap it and continue the loop
			err = u.Unwrap() // The next iteration will add the error to the result
			continue
		} else if u, ok := err.(unwrapPacked); ok {
			// The error implements `Unwrap() []error`, so just add the resulting errors to the result and break the loop
			errs := u.Unwrap()
			for _, e := range errs {
				add(Unpack(e)...)
			}
			break
		} else {
			break // nothing to unpack
		}
	}
	return result
}

type ExternalError interface {
	ExternalError() bool
}

func IsExternalError(err error) bool {
	if err == nil {
		return false
	}

	for _, err := range Unpack(err) {
		errExternal, ok := err.(ExternalError)
		if ok && errExternal.ExternalError() {
			return true
		}
	}

	return false
}
