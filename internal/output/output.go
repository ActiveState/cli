package output

import (
	"io"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// EditorV0FormatName is the format used for Komodo. We do not implement the actual formatter at this level as the
// format is by definition unstructured (ie. needs to be handled case by case)
const EditorV0FormatName = "editor.v0"

// FailNotRecognized is a failure due to the format not being recognized
var FailNotRecognized = failures.Type("output.fail.not.recognized", failures.FailInput)

// Outputer is the initialized formatter
type Outputer interface {
	Print(value interface{})
	Error(value interface{})
	Config() *Config
}

// New constructs a new Outputer according to the given format name
func New(formatName string, config *Config) (Outputer, *failures.Failure) {
	registry := NewRegistry()
	return registry.New(formatName, config)
}

// Registry holds optional additional formatters that can be defined outside of this package
// If you're not using custom formatters you can just use the regular (non-registry) constructor (New)
type Registry struct {
	registered map[string]Outputer
}

// NewRegistry constructs a new Registry. What did you expect?
func NewRegistry() Registry {
	return Registry{}
}

// New constructs a new Outputer
func (r *Registry) New(formatName string, config *Config) (Outputer, *failures.Failure) {
	logging.Debug("Requested outputer for %s", formatName)

	if handler, ok := r.registered[formatName]; ok {
		logging.Debug("Using registered outputer for %s", formatName)
		return handler, nil
	}

	switch formatName {
	case PlainFormatName:
		logging.Debug("Using Plain outputer")
		plain, fail := NewPlain(config)
		return &plain, fail
	case JSONFormatName, EditorFormatName:
		logging.Debug("Using JSON outputer")
		json, fail := NewJSON(config)
		return &json, fail
	}

	return nil, FailNotRecognized.New(locale.Tr("err_unknown_format", formatName))
}

// Register registers a custom Outputer
func (r *Registry) Register(formatName string, handler Outputer) {
	r.registered[formatName] = handler
}

// Config is the thing we pass to Outputer constructors
type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
