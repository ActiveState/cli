// +build !test

package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rollbar/rollbar-go"

	"github.com/ActiveState/cli/internal/config"
)

// Logger describes a logging function, like Debug, Error, Warning, etc.
type Logger func(msg string, args ...interface{})

type fileHandler struct {
	formatter Formatter
	file      *os.File
	verbose   bool
}

func (l *fileHandler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *fileHandler) SetVerbose(v bool) {
	l.verbose = v
}

func (l *fileHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	datadir := config.ConfigPath()
	filename := filepath.Join(datadir, "log.txt")

	if ctx.Level == "ERROR" {
		rollbar.Error(fmt.Errorf(message, args...))
	}

	message = l.formatter.Format(ctx, message, args...)
	if l.verbose {
		fmt.Fprintln(os.Stderr, message)
	}

	if l.file == nil {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		l.file = f
	}

	_, err := l.file.WriteString(message + "\n")
	if err != nil {
		return err
	}

	return nil
}

func init() {
	handler := &fileHandler{DefaultFormatter, nil, os.Getenv("VERBOSE") != ""}
	SetHandler(handler)
}
