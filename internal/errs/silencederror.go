package errs

import (
	"errors"
)

type silencedError struct {
	error
}

func Silence(err error) silencedError {
	return silencedError{err}
}

func (s *silencedError) Unwrap() error { return s.error }

func (s *silencedError) IsSilent() bool { return true }

func IsSilent(err error) bool {
	var silentErr interface {
		IsSilent() bool
	}
	return errors.As(err, &silentErr) && silentErr.IsSilent()
}
