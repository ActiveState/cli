package config

// Error is a special error to convey localization info and avoid a circular
// import.
type Error struct {
	err     error
	key     string
	baseMsg string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.err.Error()
}

// Unwrap facilitates error chain unwrapping.
func (e *Error) Unwrap() error {
	return e.err
}

// Localization implements locale.Localizer.
func (e *Error) Localization() (key, baseMsg string) {
	return e.key, e.baseMsg
}
