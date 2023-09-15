package errs

import "errors"

type UserFacingError interface {
	error
	UserError() string
}

type ErrOpt func(err *userFacingError) *userFacingError

type userFacingError struct {
	wrapped error
	message string
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

func NewUserFacingError(message string, tips ...string) *userFacingError {
	return WrapUserFacingError(nil, message)
}

func WrapUserFacingError(wrapTarget error, message string, opts ...ErrOpt) *userFacingError {
	err := &userFacingError{
		wrapTarget,
		message,
		nil,
	}

	for _, opt := range opts {
		err = opt(err)
	}

	return err
}

func IsUserFacing(err error) bool {
	var userFacingError UserFacingError
	return errors.As(err, &userFacingError)
}

func WithTips(tips ...string) ErrOpt {
	return func(err *userFacingError) *userFacingError {
		err.tips = append(err.tips, tips...)
		return err
	}
}
