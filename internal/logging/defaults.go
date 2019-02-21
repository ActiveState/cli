// +build !test

package logging

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
)

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
	datadir := config.GetDataDir()
	filename := filepath.Join(datadir, "log.txt")

	if l.verbose {
		fmt.Fprintln(os.Stderr, l.formatter.Format(ctx, message, args...))
	}

	if l.file == nil {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		l.file = f
	}

	_, err := l.file.WriteString(l.formatter.Format(ctx, message, args...) + "\n")
	if err != nil {
		return err
	}

	return nil
}

func init() {
	handler := &fileHandler{DefaultFormatter, nil, flag.Lookup("test.v") != nil}
	SetHandler(handler)
}
