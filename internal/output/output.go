package output

import (
	"io"

	"github.com/ActiveState/cli/internal/failures"
)

const EditorV0FormatName = "editor.v0"

type Outputer interface {
	Print(value interface{})
	Error(value interface{})
}

func New(formatName string, config *Config) (Outputer, *failures.Failure) {
	registry := NewRegistry()
	return registry.New(formatName, config)
}

type Registry struct {
	registered map[string]Outputer
}

func NewRegistry() Registry {
	return Registry{}
}

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

func (r *Registry) Register(formatName string, handler Outputer) {
	r.registered[formatName] = handler
}

type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
