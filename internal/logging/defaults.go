// +build !test

package logging

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/rollbar/rollbar-go"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
)

// datadir is the base directory at which the log is saved
var datadir string

var timestamp int64

// CurrentCmd holds the value of the current command being invoked
// it's a quick hack to allow us to log the command to rollbar without risking exposing sensitive info
var CurrentCmd string

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
	return FilePathFor(FileName())
}

func FilePathFor(filename string) string {
	return filepath.Join(datadir, "logs", filename)
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

	filename := FilePath()
	originalMessage := fmt.Sprintf(message, args...)

	// only log to rollbar when on release, beta or unstable branch and when built via CI (ie., non-local build)
	defer func() { // defer so that we can ensure errors are logged to the logfile even if rollbar panics (which HAS happened!)
		if (ctx.Level == "ERROR" || ctx.Level == "CRITICAL") && (constants.BranchName == constants.ReleaseBranch || constants.BranchName == constants.BetaBranch || constants.BranchName == constants.ExperimentalBranch) && rtutils.BuiltViaCI {
			data := map[string]interface{}{}

			if l.file != nil {
				if err := l.file.Close(); err != nil {
					data["log_file_close_error"] = err.Error()
				} else {
					logDatab, err := ioutil.ReadFile(filename)
					if err != nil {
						data["log_file_read_error"] = err.Error()
					} else {
						logData := string(logDatab)
						if len(logData) > 5000 {
							logData = "<truncated>\n" + logData[len(logData)-5000:]
						}
						data["log_file_data"] = logData
					}
				}
				l.file = nil // unset so that it is reset later in this func
			}

			exec := CurrentCmd
			if exec == "" {
				exec = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
			}
			flags := []string{}
			for _, arg := range os.Args[1:] {
				if strings.HasPrefix(arg, "-") {
					idx := strings.Index(arg, "=")
					if idx != -1 {
						arg = arg[0:idx]
					}
					flags = append(flags, arg)
				}
			}

			rollbarMsg := fmt.Sprintf("%s %s: %s", exec, flags, originalMessage)
			if len(rollbarMsg) > 1000 {
				rollbarMsg = rollbarMsg[0:1000] + " <truncated>"
			}

			if ctx.Level == "CRITICAL" {
				rollbar.Critical(errs.New(rollbarMsg), data)
			} else {
				rollbar.Error(errs.New(rollbarMsg), data)
			}
		}
	}()

	message = l.formatter.Format(ctx, message, args...)
	if l.verbose.value() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("(PID %d) %s", os.Getpid(), message))
	}

	if l.file == nil {
		if err := l.reopenLogfile(); err != nil {
			return errs.Wrap(err, "Failed to reopen log-file")
		}

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
		// try to reopen the log file once:
		if rerr := l.reopenLogfile(); rerr != nil {
			return errs.Wrap(err, "Failed to write log line and reopen failed with err: %v", rerr)
		}
		if _, err2 := l.file.WriteString(message + "\n"); err2 != nil {
			return errs.Wrap(err2, "Failed to write log line twice. First error was: %v", err)
		}
	}

	return nil
}

// Printf satifies a Logger interface allowing us to funnel our
// logging handlers to 3rd party libraries
func (l *fileHandler) Printf(msg string, args ...interface{}) {
	logMsg := fmt.Sprintf("Third party log message: %s", msg)
	l.Emit(getContext("DEBUG", 1), logMsg, args...)
}

func (l *fileHandler) reopenLogfile() error {
	filename := FilePath()
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return errs.Wrap(err, "Could not ensure dir exists")
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return errs.Wrap(err, "Could not open log file for writing: %s", filename)
	}
	l.file = f
	return nil
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
	files = funk.Filter(files, func(f fs.FileInfo) bool {
		return f.ModTime().Before(time.Now().Add(-time.Hour))
	}).([]fs.FileInfo)

	c := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), FileNamePrefix()) && strings.HasSuffix(file.Name(), FileNameSuffix) {
			c = c + 1
			if c > 9 {
				if err := os.Remove(filepath.Join(datadir, file.Name())); err != nil {
					Error("Could not clean up old log: %s, error: %v", file.Name(), err)
				}
			}
		}
	}

	Debug("Args: %v", os.Args)
}
