package output

import (
	"fmt"
	"io"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

// Simple is an outputer that works exactly as Plain without
// any notice level output
type Simple struct {
	cfg *Config
}

// NewSimple constructs a new Simple struct
func NewSimple(config *Config) (Simple, *failures.Failure) {
	return Simple{config}, nil
}

// Type tells callers what type of outputer we are
func (s *Simple) Type() Format {
	return SimpleFormatName
}

// Print will marshal and print the given value to the output writer
func (s *Simple) Print(value interface{}) {
	s.write(s.cfg.OutWriter, value)
	s.write(s.cfg.OutWriter, "\n")
}

// Error will marshal and print the given value to the error writer, it wraps it in the error format but otherwise the
// only thing that identifies it as an error is the channel it writes it to
func (s *Simple) Error(value interface{}) {
	s.write(s.cfg.ErrWriter, fmt.Sprintf("[ERROR]%s[/RESET]\n", value))
}

// Notice has no effect for this outputer
func (s *Simple) Notice(value interface{}) {}

// Config returns the Config struct for the active instance
func (s *Simple) Config() *Config {
	return s.cfg
}

func (s *Simple) write(writer io.Writer, value interface{}) {
	v, err := sprint(value)
	if err != nil {
		logging.Errorf("Could not sprint value: %v, error: %v, stack: %s", value, err, stacktrace.Get().String())
		writeNow(s.cfg.ErrWriter, s.cfg.Colored, fmt.Sprintf("[ERROR]%s[/RESET]", locale.Tr("err_sprint", err.Error())))
		return
	}
	writeNow(writer, s.cfg.Colored, v)
}
