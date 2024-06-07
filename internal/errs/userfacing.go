package errs

import (
	"errors"
)

type UserFacingError interface {
	error
	UserError() string
}

type ErrOpt func(err *userFacingError)

type userFacingError struct {
	wrapped error
	message string
	input   bool
	tips    []string
}

func (e *userFacingError) Error() string {
	return "User Facing Error: " + e.UserError()
}

func (e *userFacingError) UserError() string {
	return e.message
}

func (e *userFacingError) ErrorTips() []string {
	return e.tips
}

func (e *userFacingError) InputError() bool {
	return e.input
}

func (e *userFacingError) Unwrap() error {
	return e.wrapped
}

func NewUserFacing(message string, opts ...ErrOpt) *userFacingError {
	return WrapUserFacing(nil, message, opts...)
}

func WrapUserFacing(wrapTarget error, message string, opts ...ErrOpt) *userFacingError {
	err := &userFacingError{
		wrapTarget,
		message,
		false,
		nil,
	}

	for _, opt := range opts {
		opt(err)
	}

	return err
}

func IsUserFacing(err error) bool {
	var userFacingError UserFacingError
	return errors.As(err, &userFacingError)
}

// SetIf is a helper for setting options if some conditional evaluated to true.
// This is mainly intended for setting tips, as without this you'd have to evaluate your conditional outside of
// NewUserFacing/WrapUserFacing, adding to the boilerplate.
func SetIf(evaluated bool, opt ErrOpt) ErrOpt {
	if evaluated {
		return opt
	}
	return func(err *userFacingError) {}
}

func SetTips(tips ...string) ErrOpt {
	return func(err *userFacingError) {
		err.tips = append(err.tips, tips...)
	}
}

func SetInput() ErrOpt {
	return func(err *userFacingError) {
		err.input = true
	}
}
