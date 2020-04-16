package errs

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

type WrappedError struct {
	error
	wrapped error
}

func (e *WrappedError) Unwrap() error {
	return e.wrapped
}

type LocalizedError struct {
	error
}

func (e *LocalizedError) Localized() bool {
	return true
}

type Localizer interface {
	Localized() bool
}

func New(message string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(message, args...))
}

func NewWrapped(err error, message string, args ...interface{}) error {
	return &WrappedError{
		New(message, args...),
		err,
	}
}

func NewLocalized(locale string) error {
	return &LocalizedError{New(locale)}
}

func NewError(err error) error {
	return NewErrorWithLogger(err, logging.Error)
}

// NewErrorWithLogger is used by tests primarily, so we can unit test the logging part
// perhaps this would live better in a struct so we can not rely on globals at all, but for now that's not worth the effort
func NewErrorWithLogger(err error, logger func(v string, args ...interface{})) error {
	stack := stacktrace.Get()
	logger("Error created: %s. Stack:\n%s", err.Error(), stack.String())
	return err
}

func Localize(err error, locale string) error {
	return &LocalizedError{&WrappedError{errors.New(locale), err}}
}

func IsLocale(err error) bool {
	var errLocale Localizer = &LocalizedError{}
	return errors.As(err, &errLocale)
}
