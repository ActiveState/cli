// A simple logging module that mimics the behavior of Python's logging module.
//
// All it does basically is wrap Go's logger with nice multi-level logging calls, and
// allows you to set the logging level of your app in runtime.
//
// Logging is done just like calling fmt.Sprintf:
// 		logging.Info("This object is %s and that is %s", obj, that)
//
// example output:
//	2013/05/07 01:20:26 INFO @ db.go:528: Registering plugin REPLICATION
//	2013/05/07 01:20:26 INFO @ db.go:562: Registered 6 plugins and 22 commands
//	2013/05/07 01:20:26 INFO @ slave.go:277: Running replication watchdog loop!
//	2013/05/07 01:20:26 INFO @ redis.go:49: Redis adapter listening on 0.0.0.0:2000
//	2013/05/07 01:20:26 WARN @ main.go:69: Starting adapter...
//	2013/05/07 01:20:26 INFO @ db.go:966: Finished dump load. Loaded 2 objects from dump
//	2013/05/07 01:22:26 INFO @ db.go:329: Checking persistence... 0 changes since 2m0.000297531s
//	2013/05/07 01:22:26 INFO @ db.go:337: No need to save the db. no changes...
//	2013/05/07 01:22:26 DEBUG @ db.go:341: Sleeping for 2m0s
//
package logging

// This package may NOT depend on failures (directly or indirectly)

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

const (
	DEBUG    = 1
	INFO     = 2
	WARNING  = 4
	WARN     = 4
	ERROR    = 8
	NOTICE   = 16 //notice is like info but for really important stuff ;)
	CRITICAL = 32
	QUIET    = ERROR | NOTICE | CRITICAL               //setting for errors only
	NORMAL   = INFO | WARN | ERROR | NOTICE | CRITICAL // default setting - all besides debug
	ALL      = 255
	NOTHING  = 0
)

var levels_ascending = []int{DEBUG, INFO, WARNING, ERROR, NOTICE, CRITICAL}

var LevlelsByName = map[string]int{
	"DEBUG":    DEBUG,
	"INFO":     INFO,
	"WARNING":  WARN,
	"WARN":     WARN,
	"ERROR":    ERROR,
	"NOTICE":   NOTICE,
	"CRITICAL": CRITICAL,
	"QUIET":    QUIET,
	"NORMAL":   NORMAL,
	"ALL":      ALL,
	"NOTHING":  NOTHING,
}

//default logging level is ALL
var level int = ALL

// Set the logging level.
//
// Contrary to Python that specifies a minimal level, this logger is set with a bit mask
// of active levels.
//
// e.g. for INFO and ERROR use:
// 		SetLevel(logging.INFO | logging.ERROR)
//
// For everything but debug and info use:
// 		SetLevel(logging.ALL &^ (logging.INFO | logging.DEBUG))
//
func SetLevel(l int) {
	level = l
}

// Set a minimal level for loggin, setting all levels higher than this level as well.
//
// the severity order is DEBUG, INFO, WARNING, ERROR, CRITICAL
func SetMinimalLevel(l int) {

	newLevel := 0
	for _, level := range levels_ascending {
		if level >= l {
			newLevel |= level
		}
	}
	SetLevel(newLevel)

}

// Set minimal level by string, useful for config files and command line arguments. Case insensitive.
//
// Possible level names are DEBUG, INFO, WARNING, ERROR, NOTICE, CRITICAL
func SetMinimalLevelByName(l string) error {
	l = strings.ToUpper(strings.Trim(l, " "))
	level, found := LevlelsByName[l]
	if !found {
		Error("Could not set level - not found level %s", l)
		return fmt.Errorf("Invalid level %s", l)
	}

	SetMinimalLevel(level)
	return nil
}

// Set the output writer. for now it just wraps log.SetOutput()
func SetOutput(w io.Writer) {
	log.SetOutput(w)
}

//a pluggable logger interface
type LoggingHandler interface {
	SetFormatter(Formatter)
	SetVerbose(bool)
	SetConfig(cfg config)
	Output() io.Writer
	Emit(ctx *MessageContext, message string, args ...interface{}) error
	Printf(msg string, args ...interface{})
	Close()
}

type strandardHandler struct {
	formatter Formatter
}

func (l *strandardHandler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *strandardHandler) SetVerbose(v bool) {
}

func (l *strandardHandler) SetConfig(cfg config) {}

func (l *strandardHandler) Output() io.Writer {
	return nil
}

// default handling interface - just
func (l *strandardHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	fmt.Fprintln(os.Stderr, l.formatter.Format(ctx, message, args...))
	return nil
}

// Printf satifies a Logger interface allowing us to funnel our
// logging handlers to 3rd party libraries
func (l *strandardHandler) Printf(msg string, args ...interface{}) {
	logMsg := fmt.Sprintf("Third party log message: %s", msg)
	l.Emit(getContext("DEBUG", 1), logMsg, args...)
}

func (l *strandardHandler) Close() {}

var currentHandler LoggingHandler = &strandardHandler{
	DefaultFormatter,
}

// Set the current handler of the library. We currently support one handler, but it might be nice to have more
func SetHandler(h LoggingHandler) {
	currentHandler = h
}

func CurrentHandler() LoggingHandler {
	return currentHandler
}

type MessageContext struct {
	Level     string
	File      string
	Line      int
	TimeStamp time.Time
}

//get the stack (line + file) context to return the caller to the log
func getContext(level string, skipDepth int) *MessageContext {

	_, file, line, _ := runtime.Caller(skipDepth)
	file = path.Base(file)

	return &MessageContext{
		Level:     level,
		File:      file,
		TimeStamp: time.Now(),
		Line:      line,
	}
}

// Output debug logging messages
func Debug(msg string, args ...interface{}) {
	if level&DEBUG != 0 {
		writeMessage("DEBUG", msg, args...)
	}
}

type writer struct{}

func (w *writer) Write(p []byte) (n int, err error) {
	if level&DEBUG != 0 {
		writeMessage("DEBUG", string(p))
	}
	return len(p), nil
}

// format the message
func writeMessage(level string, msg string, args ...interface{}) {
	writeMessageDepth(4, level, msg, args...)
}

func writeMessageDepth(depth int, level string, msg string, args ...interface{}) {
	ctx := getContext(level, depth)

	// We go over the args, and replace any function pointer with the signature
	// func() interface{} with the return value of executing it now.
	// This allows lazy evaluation of arguments which are return values
	for i, arg := range args {
		switch arg.(type) {
		case func() interface{}:
			args[i] = arg.(func() interface{})()
		default:

		}
	}

	err := currentHandler.Emit(ctx, msg, args...)
	if err != nil {
		printLogError(err, ctx, msg, args...)

	}

}

func printLogError(err error, ctx *MessageContext, msg string, args ...interface{}) {
	errMsg := err.Error()
	errw := err
	for {
		errw = errors.Unwrap(errw)
		if errw == nil {
			break
		}
		errMsg += ": " + errw.Error()
	}
	fmt.Fprintf(os.Stderr, "Error writing log message: %s\n", errMsg)
	fmt.Fprintln(os.Stderr, DefaultFormatter.Format(ctx, msg, args...))
}

//output INFO level messages
func Info(msg string, args ...interface{}) {

	if level&INFO != 0 {

		writeMessage("INFO", msg, args...)

	}
}

// Output WARNING level messages
func Warning(msg string, args ...interface{}) {
	if level&WARN != 0 {
		writeMessage("WARNING", msg, args...)
	}
}

// Same as Warning() but return a formatted error object, regardless of logging level
func Warningf(msg string, args ...interface{}) error {
	err := fmt.Errorf(msg, args...)
	if level&WARN != 0 {
		writeMessage("WARNING", err.Error())
	}

	return err
}

// Output ERROR level messages
func Error(msg string, args ...interface{}) {
	if level&ERROR != 0 {
		writeMessage("ERROR", msg+"\n\nStacktrace: "+stacktrace.Get().String()+"\n", args...)
	}
}

// Same as Error() but also returns a new formatted error object with the message regardless of logging level
func Errorf(msg string, args ...interface{}) error {
	err := fmt.Errorf(msg, args...)
	if level&ERROR != 0 {
		writeMessage("ERROR", err.Error())
	}
	return err
}

// Output NOTICE level messages
func Notice(msg string, args ...interface{}) {
	if level&NOTICE != 0 {
		writeMessage("NOTICE", msg, args...)
	}
}

// Output a CRITICAL level message while showing a stack trace
func Critical(msg string, args ...interface{}) {
	if level&CRITICAL != 0 {
		writeMessage("CRITICAL", msg, args...)
		log.Println(string(debug.Stack()))
	}
}

// Same as critical but also returns an error object with the message regardless of logging level
func Criticalf(msg string, args ...interface{}) error {

	err := fmt.Errorf(msg, args...)
	if level&CRITICAL != 0 {
		writeMessage("CRITICAL", err.Error())
		log.Println(string(debug.Stack()))
	}
	return err
}

// Raise a PANIC while writing the stack trace to the log
func Panic(msg string, args ...interface{}) {
	log.Println(string(debug.Stack()))
	log.Panicf(msg, args...)

}

func Close() {
	currentHandler.Close()
}

func init() {
	log.SetFlags(0)
}

// bridge bridges the logger and the default go log, with a given level
type bridge struct {
	level     int
	levelName string
}

func (lb bridge) Write(p []byte) (n int, err error) {
	if level&lb.level != 0 {
		writeMessageDepth(6, lb.levelName, string(bytes.TrimRight(p, "\r\n")))
	}
	return len(p), nil
}

// BridgeStdLog bridges all messages written using the standard library's log.Print* and makes them output
// through this logger, at a given level.
func BridgeStdLog(level int) {

	for k, l := range LevlelsByName {
		if l == level {
			b := bridge{
				level:     l,
				levelName: k,
			}

			log.SetOutput(b)
		}
	}
}

func handlePanics(err interface{}) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Failed to log error. Please report this on the forums if it keeps happening. Error: %v\n", err)
}
