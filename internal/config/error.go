package config

// LocLogError conveys error info needed while avoiding a circular import.
type LocLogError struct {
	Err       error
	Key       string
	BaseMsg   string
	ReportMsg string
}

// Error implements the error interface.
func (e *LocLogError) Error() string {
	return e.Err.Error()
}

// Unwrap facilitates error chain unwrapping.
func (e *LocLogError) Unwrap() error {
	return e.Err
}

// Localization returns l10n info.
func (e *LocLogError) Localization() (key, baseMsg string) {
	return e.Key, e.BaseMsg
}

// ReportMessage is the message that should be tracked via remote logging.
func (e *LocLogError) ReportMessage() string {
	return e.ReportMsg
}
