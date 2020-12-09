package output

import (
	"fmt"
	"io"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

type Simple struct {
	cfg *Config
}

func NewSimple(config *Config) (Simple, *failures.Failure) {
	return Simple{config}, nil
}

func (s *Simple) Type() Format {
	return SimpleFormatName
}

func (s *Simple) Print(value interface{}) {
	s.write(s.cfg.OutWriter, value)
	s.write(s.cfg.OutWriter, "\n")
}

func (s *Simple) Error(value interface{}) {
	s.write(s.cfg.ErrWriter, fmt.Sprintf("[ERROR]%s[/RESET]\n", value))
}

func (s *Simple) Notice(value interface{}) {}

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
