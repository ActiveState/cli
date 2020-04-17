package locale

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
)

type LocalizedError struct {
	error
}

func (e *LocalizedError) Localized() bool {
	return true
}

type ErrorLocalizer interface {
	Localized() bool
}

func NewError(id, locale string, args ...string) error {
	translation := Tl(id, locale, args...)
	return &LocalizedError{errs.New(translation)}
}

func WrapError(err error, id, locale string, args ...string) error {
	translation := Tl(id, locale, args...)
	return &LocalizedError{errs.Wrap(errs.New(translation), err)}
}

func IsError(err error) bool {
	var errLocale ErrorLocalizer = &LocalizedError{}
	return errors.As(err, &errLocale)
}

// JoinErrors joins all error messages in the Unwrap stack that are localized
func JoinErrors(err error, sep string) error {
	var message []string
	for err != nil {
		if _, ok := err.(ErrorLocalizer); ok {
			message = append(message, err.Error())
		}
		err = errors.Unwrap(err)
	}
	return errs.Wrap(NewError("", strings.Join(message, sep)), err)
}
