package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var defaultMaxEntries = 1000

type entry struct {
	ctx     *MessageContext
	message string
	args    []interface{}
}

type fileHandler struct {
	formatter Formatter
	file      *os.File
	mu        sync.Mutex
	verbose   safeBool
	wg        *sync.WaitGroup
	queue     chan entry
	quit      chan struct{}
	closed    bool
}

func newFileHandler() *fileHandler {
	handler := fileHandler{
		DefaultFormatter,
		nil,
		sync.Mutex{},
		safeBool{},
		&sync.WaitGroup{},
		make(chan entry, defaultMaxEntries),
		make(chan struct{}),
		false,
	}
	handler.wg.Add(1)
	go func() {
		defer handler.wg.Done()
		handler.start()
	}()
	return &handler
}

func (l *fileHandler) start() {
	defer func() { handlePanics(recover()) }()

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

	message = l.formatter.Format(ctx, message, args...)
	if l.verbose.value() {
		fmt.Fprintf(os.Stderr, "(PID %d) %s\n", os.Getpid(), message)
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
	if l.closed {
		return
	}

	close(l.quit)
	l.wg.Wait()

	if l.file != nil {
		err := l.file.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close log file. Error: %v\n", err)
		}
	}

	l.closed = true
}
