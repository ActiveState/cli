package output

import (
	"io"

	"github.com/ActiveState/cli/internal/failures"
)

// EditorV0FormatName is the format used for Komodo. We do not implement the actual formatter at this level as the
// format is by definition unstructured (ie. needs to be handled case by case)
const EditorV0FormatName = "editor.v0"

// Outputer is the initialized formatter
type Outputer interface {
	Print(value interface{})
	Error(value interface{})
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
	if handler, ok := r.registered[formatName]; ok {
		return handler, nil
	}

	switch formatName {
	case PlainFormatName:
		plain, fail := NewPlain(config)
		return &plain, fail
	case JSONFormatName, EditorFormatName:
		json, fail := NewJSON(config)
		return &json, fail
	}

	return nil, nil
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
