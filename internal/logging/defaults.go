// +build !test

package logging

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
)

type fileHandler struct {
	formatter Formatter
	file      *os.File
}

func (l *fileHandler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *fileHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	datadir := config.GetDataDir()
	filename := filepath.Join(datadir, "log.txt")

	if l.file == nil {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, os.ModePerm)
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
	handler := &fileHandler{DefaultFormatter, nil}
	SetHandler(handler)
}
