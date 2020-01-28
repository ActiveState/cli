package output

import (
	"io"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
)

// Outputer is the initialized formatter
type Outputer interface {
	Print(value interface{})
	Error(value interface{})
	Config() *Config
}

// New constructs a new Outputer according to the given format name
func New(format Format, config *Config) (Outputer, *failures.Failure) {
	logging.Debug("Requested outputer for %s", format)

	switch format {
	case FormatJSON, FormatEditor:
		logging.Debug("Using %s outputer", format.String())
		json, fail := NewJSON(config)
		return &json, fail
	case FormatPlain:
	default:
		logging.Debug("Unrecognized/unset outputer format")
		format = FormatPlain
	}

	logging.Debug("Using %s outputer", format.String())
	plain, fail := NewPlain(config)
	return &plain, fail
}

// Config is the thing we pass to Outputer constructors
type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
