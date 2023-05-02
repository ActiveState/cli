package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
)

// JSON is our JSON outputer, there's not much going on here, just forwards it to the JSON marshaller and provides
// a basic structure for error
type JSON struct {
	cfg      *Config
	printNUL bool
}

// NewJSON constructs a new JSON struct
func NewJSON(config *Config) (JSON, error) {
	return JSON{config, true}, nil
}

// Type tells callers what type of outputer we are
func (f *JSON) Type() Format {
	return JSONFormatName
}

// Print will marshal and print the given value to the output writer
func (f *JSON) Print(v interface{}) {
	f.Fprint(f.cfg.OutWriter, v)
}

// Fprint allows printing to a specific writer, using all the conveniences of the output package
func (f *JSON) Fprint(writer io.Writer, value interface{}) {
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

	writer.Write(b)

	var nul string
	if f.printNUL {
		nul = "\x00"
	}
	f.cfg.OutWriter.Write([]byte(nul + "\n")) // Terminate with NUL character so consumers can differentiate between multiple output messages
}

type StructuredError struct {
	Errors []string `json:"errors"`
	Code   int      `json:"code"`
}

// Error will marshal and print the given value to the error writer
// NOTE that JSON always prints to the output writer, the error writer is unused.
func (f *JSON) Error(value interface{}) {
	var b []byte
	var err error
	switch value.(type) {
	case []byte:
		b = value.([]byte)
	case StructuredError:
		b, err = json.Marshal(value)
	default:
		b, err = json.Marshal(toStructuredError(value))
	}
	if err != nil {
		multilog.Error("Could not marshal value, error: %v", err)
		b = []byte(locale.T("err_could_not_marshal_print"))
	}
	b = []byte(colorize.StripColorCodes(string(b)))

	f.cfg.OutWriter.Write(b)

	var nul string
	if f.printNUL {
		nul = "\x00"
	}
	f.cfg.OutWriter.Write([]byte(nul + "\n")) // Terminate with NUL character so consumers can differentiate between multiple output messages
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

// toStructuredError attempts to convert the given interface into a StructuredError struct.
// It accepts an error object or a single string error message.
// If it cannot perform the conversion, it returns a StructuredError indicating so.
func toStructuredError(v interface{}) StructuredError {
	switch v.(type) {
	case error:
		return StructuredError{[]string{v.(error).Error()}, 1}
	case string:
		return StructuredError{[]string{v.(string)}, 1}
	}
	message := fmt.Sprintf("Not a recognized error format: %v", v)
	multilog.Error(message)
	return StructuredError{[]string{message}, 1}
}
