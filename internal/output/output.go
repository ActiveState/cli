package output

import (
	"io"

	"github.com/ActiveState/cli/internal/failures"
)

type Outputer interface {
	Print(value interface{})
	Error(value interface{})
	Close() error
}

func New(formatName string, config *Config) (Outputer, *failures.Failure) {
	switch formatName {
	case PlainFormatName:
		plain, fail := NewPlain(config)
		return &plain, fail
	}

	return nil, nil
}

type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
