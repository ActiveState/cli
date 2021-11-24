package output

import (
	"io"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Format string

// FormatName constants are tokens representing supported output formats.
const (
	PlainFormatName    Format = "plain"     // human readable
	SimpleFormatName   Format = "simple"    // human readable without notice level
	JSONFormatName     Format = "json"      // plain json
	EditorFormatName   Format = "editor"    // alias of "json"
	EditorV0FormatName Format = "editor.v0" // for Komodo: alias of "json"
)

// Behavior defines control tokens that affect printing behavior.
type Behavior int

// Behavior tokens.
const (
	Suppress Behavior = iota
)

var ErrNotRecognized = errs.New("Not Recognized")

// Outputer is the initialized formatter
type Outputer interface {
	Fprint(writer io.Writer, value interface{})
	Print(value interface{})
	Error(value interface{})
	Notice(value interface{})
	Type() Format
	Config() *Config
}

// lastCreated is here for specific legacy use cases
var lastCreated Outputer

// New constructs a new Outputer according to the given format name
func New(formatName string, config *Config) (Outputer, error) {
	var err error
	lastCreated, err = new(formatName, config)
	return lastCreated, err
}

func new(formatName string, config *Config) (Outputer, error) {
	logging.Debug("Requested outputer for %s", formatName)

	format := Format(formatName)
	switch format {
	case "", PlainFormatName:
		logging.Debug("Using Plain outputer")
		plain, err := NewPlain(config)
		return &Mediator{&plain, PlainFormatName}, err
	case SimpleFormatName:
		logging.Debug("Using Simple outputter")
		simple, err := NewSimple(config)
		return &Mediator{&simple, SimpleFormatName}, err
	case JSONFormatName:
		logging.Debug("Using JSON outputer")
		json, err := NewJSON(config)
		return &Mediator{&json, JSONFormatName}, err
	case EditorFormatName:
		logging.Debug("Using Editor outputer")
		editor, err := NewEditor(config)
		return &Mediator{&editor, EditorFormatName}, err
	case EditorV0FormatName:
		logging.Debug("Using EditorV0 outputer")
		editor0, err := NewEditorV0(config)
		return &Mediator{&editor0, EditorV0FormatName}, err
	}

	return nil, locale.WrapInputError(ErrNotRecognized, "err_unknown_format", string(formatName))
}

// Get is here for legacy use-cases, DO NOT USE IT FOR NEW CODE
func Get() Outputer {
	return lastCreated
}

// Config is the thing we pass to Outputer constructors
type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
