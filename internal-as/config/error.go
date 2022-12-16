package config

// LocalizedError conveys error info needed while avoiding a circular import.
type LocalizedError struct {
	Err       error
	Key       string
	BaseMsg   string
	ReportMsg string
}

// Error implements the error interface.
func (e *LocalizedError) Error() string {
	return e.Err.Error()
}

// Unwrap facilitates error chain unwrapping.
func (e *LocalizedError) Unwrap() error {
	return e.Err
}

// Localization returns l10n info.
func (e *LocalizedError) Localization() (key, baseMsg string) {
	return e.Key, e.BaseMsg
}

// ReportMessage is the message that should be tracked via remote logging.
func (e *LocalizedError) ReportMessage() string {
	return e.ReportMsg
}
