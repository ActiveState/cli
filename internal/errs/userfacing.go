package errs

type UserFacingError interface {
	error
	UserError() string
}

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

func (e *userFacingError) AddTips(tips ...string) {
	e.tips = append(e.tips, tips...)
}

func (e *userFacingError) ErrorTips() []string {
	return e.tips
}

func NewUserFacingError(message string, tips ...string) *userFacingError {
	return WrapUserFacingError(nil, message)
}

func WrapUserFacingError(wrapTarget error, message string, tips ...string) *userFacingError {
	return &userFacingError{
		wrapTarget,
		message,
		tips,
	}
}
