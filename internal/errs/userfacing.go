package errs

import "errors"

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

func NewUserFacingError(message string, tips ...string) *userFacingError {
	return WrapUserFacing(nil, message)
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

func SetTips(tips ...string) ErrOpt {
	return func(err *userFacingError) {
		err.tips = append(err.tips, tips...)
	}
}

func SetInput(v bool) ErrOpt {
	return func(err *userFacingError) {
		err.input = v
	}
}
