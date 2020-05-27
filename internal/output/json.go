package output

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// JSON is our JSON outputer, there's not much going on here, just forwards it to the JSON marshaller and provides
// a basic structure for error
type JSON struct {
	cfg *Config
}

// NewJSON constructs a new JSON struct
func NewJSON(config *Config) (JSON, *failures.Failure) {
	return JSON{config}, nil
}

// Type tells callers what type of outputer we are
func (f *JSON) Type() Format {
	return JSONFormatName
}

// Print will marshal and print the given value to the output writer
func (f *JSON) Print(value interface{}) {
	b, err := json.Marshal(value)
	if err != nil {
		logging.Error("Could not marshal value, error: %v", err)
		f.Error(locale.T("err_could_not_marshal_print"))
		return
	}

	f.cfg.OutWriter.Write(b)
	f.cfg.OutWriter.Write([]byte("\x00\n")) // Terminate with NUL character so consumers can differentiate between multiple output messages
}

// Error will marshal and print the given value to the error writer, it wraps the error message in a very basic structure
// that identifies it as an error
// NOTE that JSON always prints to the output writer, the error writer is unused.
func (f *JSON) Error(value interface{}) {
	errStruct := struct{ Error interface{} }{value}
	b, err := json.Marshal(errStruct)
	if err != nil {
		logging.Error("Could not marshal value, error: %v", err)
		b = []byte(locale.T("err_could_not_marshal_print"))
	}
	f.cfg.OutWriter.Write(b)
	f.cfg.OutWriter.Write([]byte("\x00\n")) // Terminate with NUL character so consumers can differentiate between multiple output messages
}

// Notice is ignored by JSON, as they are considered as non-critical output and there's currently no reliable way to
// reliably combine this data into the eventual output
func (f *JSON) Notice(value interface{}) {
	logging.Warningf("JSON outputer truncated the following notice: %v", value)
}

// Config returns the Config struct for the active instance
func (f *JSON) Config() *Config {
	return f.cfg
}
