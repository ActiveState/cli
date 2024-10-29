package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"syscall"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
)

// JSON is our JSON outputer, there's not much going on here, just forwards it to the JSON marshaller and provides
// a basic structure for error
type JSON struct {
	cfg         *Config
	wroteOutput bool
}

// NewJSON constructs a new JSON struct
func NewJSON(config *Config) (JSON, error) {
	return JSON{cfg: config}, nil
}

// Type tells callers what type of outputer we are
func (f *JSON) Type() Format {
	return JSONFormatName
}

// Print will marshal and print the given value to the output writer
func (f *JSON) Print(v interface{}) {
	if err, isStructuredError := v.(StructuredError); isStructuredError {
		multilog.Error("Attempted to write unstructured output as json: %v", err)
		return
	}

	f.Fprint(f.cfg.OutWriter, v)
}

// Fprint allows printing to a specific writer, using all the conveniences of the output package
func (f *JSON) Fprint(writer io.Writer, value interface{}) {
	if f.wroteOutput {
		multilog.Error("Already wrote json output; skipping.")
		return
	}
	f.wroteOutput = true

	var b []byte
	if v, isBlob := value.([]byte); isBlob {
		b = v
	} else {
		value = prepareJSONValue(value)
		var err error
		b, err = json.Marshal(value)
		if err != nil {
			multilog.Error("Could not marshal value, error: %v", err)
			f.Error(locale.T("err_could_not_marshal_print"))
			return
		}
		b = []byte(colorize.StripColorCodes(string(b)))
	}

	_, err := writer.Write(b)
	if err != nil {
		if isPipeClosedError(err) {
			logging.Error("Could not write json output, error: %v", err) // do not log to rollbar
		} else {
			multilog.Error("Could not write json output, error: %v", err)
		}
	}
}

// Error will marshal and print the given value to the error writer
// NOTE that JSON always prints to the output writer, the error writer is unused.
func (f *JSON) Error(value interface{}) {
	if f.wroteOutput {
		multilog.Error("Already wrote json output; skipping.")
		return
	}
	f.wroteOutput = true

	var b []byte
	var err error
	switch value := value.(type) {
	case []byte:
		b = value
	default:
		b, err = json.Marshal(toStructuredError(value))
	}
	if err != nil {
		multilog.Error("Could not marshal value, error: %v", err)
		b = []byte(locale.T("err_could_not_marshal_print"))
	}
	b = []byte(colorize.StripColorCodes(string(b)))

	_, err = f.cfg.OutWriter.Write(b)
	if err != nil {
		if isPipeClosedError(err) {
			logging.Error("Could not write json output, error: %v", err) // do not log to rollbar
		} else {
			multilog.Error("Could not write json output, error: %v", err)
		}
	}
}

func isPipeClosedError(err error) bool {
	pipeErr := errors.Is(err, syscall.EPIPE)
	if runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(242)) {
		// Note: 232 is Windows error code ERROR_NO_DATA, "The pipe is being closed".
		// See https://go.dev/src/os/pipe_test.go
		pipeErr = true
	}
	return pipeErr
}

// Notice is ignored by JSON, as they are considered as non-critical output and there's currently no reliable way to
// reliably combine this data into the eventual output
func (f *JSON) Notice(value interface{}) {
	logging.Warning("JSON outputer truncated the following notice: %v", value)
}

// Config returns the Config struct for the active instance
func (f *JSON) Config() *Config {
	return f.cfg
}

func prepareJSONValue(v interface{}) interface{} {
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return v
}

// StructuredError communicates that an error happened due to output that was meant to be structured but wasn't.
type StructuredError struct {
	Message string   `json:"error"`
	Tips    []string `json:"tips,omitempty"`
}

func (s StructuredError) Error() string {
	return s.Message
}

// toStructuredError attempts to convert the given interface into a StructuredError struct.
// It accepts an error object or a single string error message.
// If it cannot perform the conversion, it returns a StructuredError indicating so.
func toStructuredError(v interface{}) StructuredError {
	switch vv := v.(type) {
	case StructuredError:
		return vv
	case error:
		return StructuredError{Message: locale.JoinedErrorMessage(vv)}
	case string:
		return StructuredError{Message: vv}
	}
	message := fmt.Sprintf("Not a recognized error format: %v", v)
	multilog.Error(message)
	return StructuredError{Message: message}
}
