package errs

// RollbarSkipper is an interface that when implemented by an error struct, should skip the step of reporting this error to rollbar
type RollbarSkipper interface {
	SkipRollbar()
}

// RollbarSkipError is an implementation of a RollbarSkipper
type RollbarSkipError struct {
	wrappedErr error
}

// SkipRollbar marks an error as "should not be reported to rollbar"
func SkipRollbar(err error) error {
	return &RollbarSkipError{err}
}

func (re *RollbarSkipError) Error() string {
	return "skip rollbar"
}

func (re *RollbarSkipError) Unwrap() error {
	return re.wrappedErr
}

func (re *RollbarSkipError) SkipRollbar() {}

// ShouldSkipRollbar checks if an error implements the RollbarSkipper interface
func ShouldSkipRollbar(err error) bool {
	var rs RollbarSkipper
	return Matches(err, &rs)
}
