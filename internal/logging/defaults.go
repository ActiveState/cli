// +build !test

package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rollbar/rollbar-go"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
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

func (l *fileHandler) Output() io.Writer {
	return l.file
}

func (l *fileHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	datadir := config.ConfigPath()
	filename := filepath.Join(datadir, "log.txt")

	if ctx.Level == "ERROR" && (constants.BranchName == constants.StableBranch || constants.BranchName == constants.UnstableBranch) {
		data := map[string]interface{}{}

		if l.file != nil {
			if err := l.file.Close(); err != nil {
				data["log_file_close_error"] = err.Error()
			} else {
				logData, err := ioutil.ReadFile(filename)
				if err != nil {
					data["log_file_read_error"] = err.Error()
				} else {
					data["log_file_data"] = string(logData)
				}
			}
			l.file = nil // unset so that it is reset later in this func
		}

		rollbar.Error(fmt.Errorf(message, args...), data)
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
