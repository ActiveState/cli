package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/rollbar"
)

var defaultMaxEntries = 1000

type config interface {
	GetBool(key string) bool
	IsSet(key string) bool
	Closed() bool
}

type entry struct {
	ctx     *MessageContext
	message string
	args    []interface{}
}

type fileHandler struct {
	formatter Formatter
	file      *os.File
	cfg       config
	mu        sync.Mutex
	verbose   safeBool
	wg        *sync.WaitGroup
	queue     chan entry
	quit      chan struct{}
	report    bool
}

func newFileHandler() *fileHandler {
	handler := fileHandler{
		DefaultFormatter,
		nil,
		nil,
		sync.Mutex{},
		safeBool{},
		&sync.WaitGroup{},
		make(chan entry, defaultMaxEntries),
		make(chan struct{}),
		true,
	}
	handler.wg.Add(1)
	go func() {
		defer handler.wg.Done()
		handler.start()
	}()
	return &handler
}

func (l *fileHandler) start() {
	defer handlePanics(recover())
	for {
		select {
		case entry := <-l.queue:
			l.emit(entry.ctx, entry.message, entry.args...)
		case <-l.quit:
			close(l.queue)
			for entry := range l.queue {
				l.emit(entry.ctx, entry.message, entry.args...)
			}
			return
		}
	}
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

func (l *fileHandler) SetConfig(cfg config) {
	l.cfg = cfg
	if l.cfg != nil && !l.cfg.Closed() && l.cfg.IsSet(constants.ReportErrorsConfig) {
		l.report = l.cfg.GetBool(constants.ReportErrorsConfig)
	}
}

func (l *fileHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	e := entry{
		ctx:     ctx,
		message: message,
		args:    args,
	}
	select {
	case <-l.quit:
		return nil
	default:
		l.queue <- e
	}
	return nil
}

func (l *fileHandler) emit(ctx *MessageContext, message string, args ...interface{}) {
	filename := FilePath()
	originalMessage := fmt.Sprintf(message, args...)

	// only log to rollbar when on release, beta or unstable branch and when built via CI (ie., non-local build)
	defer func() { // defer so that we can ensure errors are logged to the logfile even if rollbar panics (which HAS happened!)
		isPublicChannel := (constants.BranchName == constants.ReleaseBranch || constants.BranchName == constants.BetaBranch || constants.BranchName == constants.ExperimentalBranch)

		// All rollbar errors I observed are prefixed with "Rollbar"
		// This is meant to help guard against recursion issues
		isRollbarMsg := strings.HasPrefix(message, "Rollbar")

		if (ctx.Level == "ERROR" || ctx.Level == "CRITICAL") && l.report && isPublicChannel && !isRollbarMsg && condition.BuiltViaCI() {
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
				rollbar.Critical(fmt.Errorf(rollbarMsg), data)
			} else {
				rollbar.Error(fmt.Errorf(rollbarMsg), data)
			}
		}
	}()

	message = l.formatter.Format(ctx, message, args...)
	if l.verbose.value() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("(PID %d) %s", os.Getpid(), message))
	}

	if l.file == nil {
		if err := l.reopenLogfile(); err != nil {
			printLogError(fmt.Errorf("Failed to reopen log-file: %w", err), ctx, message, args...)
		}

		if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			printLogError(fmt.Errorf("Could not ensure dir exists: %w", err), ctx, message, args...)
		}
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			printLogError(fmt.Errorf("Could not open log file for writing: %s: %w", filename, err), ctx, message, args...)
		}
		l.file = f
	}

	_, err := l.file.WriteString(message + "\n")
	if err != nil {
		// try to reopen the log file once:
		if rerr := l.reopenLogfile(); rerr != nil {
			printLogError(fmt.Errorf("Failed to write log line and reopen failed with err: %v: %w", rerr, err), ctx, message, args...)
		}
		if _, err2 := l.file.WriteString(message + "\n"); err2 != nil {
			printLogError(fmt.Errorf("Failed to write log line twice. First error was: %v: %w", err, err2), ctx, message, args...)
		}
	}

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
		return fmt.Errorf("Could not ensure dir exists: %w", err)
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Could not open log file for writing: %s: %w", filename, err)
	}
	l.file = f
	return nil
}

func (l *fileHandler) Close() {
	close(l.quit)
	l.wg.Wait()
}
