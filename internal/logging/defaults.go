// +build !test

package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

func FileName() string {
	return FileNameFor(os.Getpid())
}

func FileNameFor(pid int) string {
	return fmt.Sprintf("%d%s", pid, FileNameSuffix)
}

func FilePath() string {
	return filepath.Join(config.ConfigPath(), FileName())
}

func FilePathFor(filename string) string {
	return filepath.Join(config.ConfigPath(), filename)
}

const FileNameSuffix = ".log"

func (l *fileHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	datadir := config.ConfigPath()
	filename := filepath.Join(datadir, FileName())

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

	// Clean up old log files
	datadir := config.ConfigPath()
	files, err := ioutil.ReadDir(datadir)
	if err != nil {
		Error("Could not scan config dir to clean up stale logs: %v", err)
		return
	}

	sort.Slice(files, func(i, j int) bool { return files[i].ModTime().After(files[j].ModTime()) })

	c := 0
	for _, file := range files {
		if strings.HasSuffix(file.Name(), FileNameSuffix) {
			c = c + 1
			if c > 9 {
				if err := os.Remove(filepath.Join(datadir, file.Name())); err != nil {
					Error("Could not clean up old log: %s, error: %v", file.Name(), err)
				}
			}
		}
	}
}
