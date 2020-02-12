package output

import (
	"encoding/json"
	"errors"

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

// Print will marshal and print the given value to the output writer
func (f *JSON) Print(value interface{}) {
	wrap := Wrap{
		Result: value,
	}

	b, err := json.Marshal(&wrap)
	if err != nil {
		logging.Error("Could not marshal value, error: %v", err)
		f.Error(locale.T("err_could_not_marshal_print"))
		return
	}

	f.cfg.OutWriter.Write(b)
	f.cfg.OutWriter.Write([]byte("\n"))
}

// Error will marshal and print the given value to the error writer, it wraps the error message in a very basic structure
// that identifies it as an error
// NOTE that JSON always prints to the output writer, the error writer is unused.
func (f *JSON) Error(value interface{}) {
	var err error
	switch v := value.(type) {
	case string:
		err = NewGeneralError(errors.New(v))
	case error:
		err = NewGeneralError(v)
	default:
		b, err := json.Marshal(value)
		if err != nil {
			logging.Error("Could not marshal value, error: %v", err)
			b = []byte(locale.T("err_could_not_marshal_print"))
		}
		err = NewGeneralError(errors.New(string(b)))
	}

	wrap := Wrap{
		Error: err,
	}

	b, err := json.Marshal(&wrap)
	if err != nil {
		logging.Error("Could not marshal value, error: %v", err)
		b = []byte(locale.T("err_could_not_marshal_print"))
	}

	f.cfg.OutWriter.Write(b)
	f.cfg.OutWriter.Write([]byte("\n"))
}

// Config returns the Config struct for the active instance
func (f *JSON) Config() *Config {
	return f.cfg
}
