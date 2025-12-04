// A simple logging module that mimics the behavior of Python's logging module.
//
// All it does basically is wrap Go's logger with nice multi-level logging calls, and
// allows you to set the logging level of your app in runtime.
//
// Logging is done just like calling fmt.Sprintf:
//
//	logging.Info("This object is %s and that is %s", obj, that)
//
// example output:
//
// [DBG 1670353253256778 instance.go:123] Setting config: projects
// [DBG 1670353253259897 subshell.go:95] Detected SHELL: zsh
// [DBG 1670353253259915 subshell.go:132] Using binary: /bin/zsh
package logging

// This package may NOT depend on failures (directly or indirectly)

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

const (
	DEBUG    = 1
	INFO     = 2
	WARNING  = 4
	WARN     = 4
	ERROR    = 8
	NOTICE   = 16 // notice is like info but for really important stuff ;)
	CRITICAL = 32
	QUIET    = ERROR | NOTICE | CRITICAL               // setting for errors only
	NORMAL   = INFO | WARN | ERROR | NOTICE | CRITICAL // default setting - all besides debug
	ALL      = 255
	NOTHING  = 0
)

var levels_ascending = []int{DEBUG, INFO, WARNING, ERROR, NOTICE, CRITICAL}

var LevelsByName = map[string]int{
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

// default logging level is ALL
var level int = ALL

// Set the logging level.
//
// Contrary to Python that specifies a minimal level, this logger is set with a bit mask
// of active levels.
//
// e.g. for INFO and ERROR use:
//
//	SetLevel(logging.INFO | logging.ERROR)
//
// For everything but debug and info use:
//
//	SetLevel(logging.ALL &^ (logging.INFO | logging.DEBUG))
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
	level, found := LevelsByName[l]
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

// a pluggable logger interface
type LoggingHandler interface {
	SetFormatter(Formatter)
	SetVerbose(bool)
	Output() io.Writer
	Emit(ctx *MessageContext, message string, args ...interface{}) error
	Printf(msg string, args ...interface{})
	Close()
}

type standardHandler struct {
	formatter Formatter
	verbose   bool
}

func (l *standardHandler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *standardHandler) SetVerbose(v bool) {
	l.verbose = v
}

func (l *standardHandler) Output() io.Writer {
	return nil
}

// default handling interface - just
func (l *standardHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	if l.verbose {
		fmt.Fprintln(os.Stderr, l.formatter.Format(ctx, message, args...))
	}
	return nil
}

// Printf satifies a Logger interface allowing us to funnel our
// logging handlers to 3rd party libraries
func (l *standardHandler) Printf(msg string, args ...interface{}) {
	logMsg := fmt.Sprintf("Third party log message: %s", msg)
	l.Emit(getContext("DBG", 1), logMsg, args...)
}

func (l *standardHandler) Close() {
	l.Emit(getContext("DEBUG", 1), "Closing logging handler")
}

var currentHandler LoggingHandler = &standardHandler{
	DefaultFormatter,
	os.Getenv("VERBOSE") != "",
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

// get the stack (line + file) context to return the caller to the log
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
		writeMessage("DBG", msg, args...)
	}
}

type writer struct{}

func (w *writer) Write(p []byte) (n int, err error) {
	if level&DEBUG != 0 {
		writeMessage("DBG", string(p))
	}
	return len(p), nil
}

// format the message
func writeMessage(level string, msg string, args ...interface{}) {
	writeMessageDepth(4, level, msg, args...)
}

// TailSize specifies the number of logged bytes to keep for use with Tail.
const TailSize = 5000

var logTail *ringBuffer
var tailLogger *log.Logger

func writeToLogTail(ctx *MessageContext, msg string, args ...interface{}) {
	if tailLogger == nil {
		logTail = newRingBuffer(TailSize)
		tailLogger = log.New(logTail, "", log.LstdFlags)
	}
	tailLogger.Println(DefaultFormatter.Format(ctx, msg, args...))
}

// ReadTail returns as a string the last TailSize bytes written by this logger.
func ReadTail() string {
	if logTail == nil {
		return ""
	}
	return logTail.Read()
}

func writeMessageDepth(depth int, level string, msg string, args ...interface{}) {
	ctx := getContext(level, depth)

	// We go over the args, and replace any function pointer with the signature
	// func() interface{} with the return value of executing it now.
	// This allows lazy evaluation of arguments which are return values
	// Also, unpack error objects.
	for i, arg := range args {
		switch arg := arg.(type) {
		case func() interface{}:
			args[i] = arg
		case error:
			args[i] = errs.JoinMessage(arg)
		default:
		}
	}

	err := currentHandler.Emit(ctx, msg, args...)
	if err != nil {
		printLogError(err, ctx, msg, args...)
	}

	writeToLogTail(ctx, msg, args...)
}

func printLogError(err error, ctx *MessageContext, msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error writing log message: %s\n", errs.JoinMessage(err))
	fmt.Fprintln(os.Stderr, DefaultFormatter.Format(ctx, msg, args...))
}

// output INFO level messages
func Info(msg string, args ...interface{}) {

	if level&INFO != 0 {

		writeMessage("INF", msg, args...)

	}
}

// Output WARNING level messages
func Warning(msg string, args ...interface{}) {
	if level&WARN != 0 {
		writeMessage("WRN", msg, args...)
	}
}

// Output ERROR level messages
// This should be used sparingly, as multilog.Error() is preferred.
func Error(msg string, args ...interface{}) {
	if level&ERROR != 0 {
		writeMessage("ERR", msg+"\n\nStacktrace: "+stacktrace.Get().String()+"\n", args...)
	}
}

// Same as Error() but without a stacktrace.
func ErrorNoStacktrace(msg string, args ...interface{}) {
	if level&ERROR != 0 {
		writeMessage("ERR", msg, args...)
	}
}

// Output NOTICE level messages
func Notice(msg string, args ...interface{}) {
	if level&NOTICE != 0 {
		writeMessage("NOT", msg, args...)
	}
}

// Output a CRITICAL level message while showing a stack trace
// This should be called sparingly, as multilog.Critical() is preferred.
func Critical(msg string, args ...interface{}) {
	if level&CRITICAL != 0 {
		writeMessage("CRT", msg, args...)
		log.Println(string(debug.Stack()))
	}
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

	for k, l := range LevelsByName {
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
