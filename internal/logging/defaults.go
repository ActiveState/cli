// +build !test

package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/rollbar/rollbar-go"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils"
)

// datadir is the base directory at which the log is saved
var datadir string

var timestamp int64

// Logger describes a logging function, like Debug, Error, Warning, etc.
type Logger func(msg string, args ...interface{})

type safeBool struct {
	mu sync.Mutex
	v  bool
}

func (s *safeBool) value() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.v
}

func (s *safeBool) setValue(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v = v
}

type fileHandler struct {
	formatter Formatter
	file      *os.File
	mu        sync.Mutex
	verbose   safeBool
}

func (l *fileHandler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *fileHandler) SetVerbose(v bool) {
	l.verbose.setValue(v)
}

func (l *fileHandler) Output() io.Writer {
	return l.file
}

func FileName() string {
	return FileNameFor(os.Getpid())
}

func FileNameFor(pid int) string {
	return FileNameForCmd(FileNamePrefix(), pid)
}

func FileNameForCmd(cmd string, pid int) string {
	if cmd == constants.StateInstallerCmd {
		return fmt.Sprintf("%s-%d%s", cmd, pid, FileNameSuffix)
	}
	return fmt.Sprintf("%s-%d-%d%s", cmd, pid, timestamp, FileNameSuffix)
}

func FileNamePrefix() string {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	exe = filepath.Base(exe)
	return strings.TrimSuffix(exe, filepath.Ext(exe))
}

func FilePath() string {
	return filepath.Join(datadir, FileName())
}

func FilePathFor(filename string) string {
	return filepath.Join(datadir, filename)
}

func FilePathForCmd(cmd string, pid int) string {
	return FilePathFor(FileNameForCmd(cmd, pid))
}

const FileNameSuffix = ".log"

func (l *fileHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	defer handlePanics(recover())
	// In this function we close and open the file handle to the log file. In
	// order to ensure this is safe to be called across threads, we just
	// synchronize the entire function
	l.mu.Lock()
	defer l.mu.Unlock()

	filename := filepath.Join(datadir, FileName())
	originalMessage := message

	// only log to rollbar when on release, beta or unstable branch and when built via CI (ie., non-local build)
	defer func() { // defer so that we can ensure errors are logged to the logfile even if rollbar panics (which HAS happened!)
		if ctx.Level == "ERROR" && (constants.BranchName == constants.ReleaseBranch || constants.BranchName == constants.BetaBranch || constants.BranchName == constants.ExperimentalBranch) && rtutils.BuiltViaCI {
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

			rollbar.Error(fmt.Errorf(originalMessage, args...), data)
		}
	}()

	message = l.formatter.Format(ctx, message, args...)
	if l.verbose.value() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("(PID %d) %s", os.Getpid(), message))
	}

	if l.file == nil {
		if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			return errs.Wrap(err, "Could not ensure dir exists")
		}
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return errs.Wrap(err, "Could not open log file for writing: %s", filename)
		}
		l.file = f
	}

	_, err := l.file.WriteString(message + "\n")
	if err != nil {
		return err
	}

	return nil
}

// Printf satifies a Logger interface allowing us to funnel our
// logging handlers to 3rd party libraries
func (l *fileHandler) Printf(msg string, args ...interface{}) {
	logMsg := fmt.Sprintf("Third party log message: %s", msg)
	l.Emit(getContext("DEBUG", 1), logMsg, args...)
}

func init() {
	defer handlePanics(recover())
	timestamp = time.Now().UnixNano()
	handler := &fileHandler{DefaultFormatter, nil, sync.Mutex{}, safeBool{}}
	SetHandler(handler)

	log.SetOutput(&writer{})

	// Clean up old log files
	var err error
	datadir, err = storage.AppDataPath()
	if err != nil {
		Error("Could not detect AppData dir: %v", err)
		return
	}

	files, err := ioutil.ReadDir(datadir)
	if err != nil && !os.IsNotExist(err) {
		Error("Could not scan config dir to clean up stale logs: %v", err)
		return
	}

	sort.Slice(files, func(i, j int) bool { return files[i].ModTime().After(files[j].ModTime()) })

	c := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), FileNamePrefix()) && strings.HasSuffix(file.Name(), FileNameSuffix) {
			c = c + 1
			if c > 19 {
				if err := os.Remove(filepath.Join(datadir, file.Name())); err != nil {
					Error("Could not clean up old log: %s, error: %v", file.Name(), err)
				}
			}
		}
	}

	Debug("Args: %v", os.Args)
}
