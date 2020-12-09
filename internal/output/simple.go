package output

import (
	"github.com/ActiveState/cli/internal/failures"
)

// Simple is an outputer that works exactly as Plain without
// any notice level output
type Simple struct {
	Plain
}

// NewSimple constructs a new Simple struct
func NewSimple(config *Config) (Simple, *failures.Failure) {
	plain, fail := NewPlain(config)
	if fail != nil {
		return Simple{}, fail
	}

	return Simple{plain}, nil
}

// Type tells callers what type of outputer we are
func (s *Simple) Type() Format {
	return SimpleFormatName
}

// Notice has no effect for this outputer
func (s *Simple) Notice(value interface{}) {}
