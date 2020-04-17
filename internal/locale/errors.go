package locale

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
)

type LocalizedError struct {
	error
}

func (e *LocalizedError) Localized() bool {
	return true
}

type Localizer interface {
	Localized() bool
}

func NewError(id, locale string, args ...string) error {
	translation := Tl(id, locale, args...)
	return &LocalizedError{errs.New(translation)}
}

func IsError(err error) bool {
	var errLocale Localizer = &LocalizedError{}
	return errors.As(err, &errLocale)
}
