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

func New(formatName string, outWriter, errWriter io.Writer) (Outputer, *failures.Failure) {
	switch formatName {
	case PlainFormatName:
		return NewPlain(outWriter, errWriter)
	}

	return nil, nil
}
