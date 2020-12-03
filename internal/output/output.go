package output

import (
	"io"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Format string

// FormatName constants are tokens representing supported output formats.
const (
	PlainFormatName    Format = "plain"     // human readable
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

// FailNotRecognized is a failure due to the format not being recognized
var FailNotRecognized = failures.Type("output.fail.not.recognized", failures.FailInput)

// Outputer is the initialized formatter
type Outputer interface {
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
	var fail error
	lastCreated, fail = new(formatName, config)
	return lastCreated, fail
}

func new(formatName string, config *Config) (Outputer, error) {
	logging.Debug("Requested outputer for %s", formatName)

	format := Format(formatName)
	switch format {
	case "", PlainFormatName:
		logging.Debug("Using Plain outputer")
		plain, fail := NewPlain(config)
		return &Mediator{&plain, PlainFormatName}, fail
	case JSONFormatName:
		logging.Debug("Using JSON outputer")
		json, fail := NewJSON(config)
		return &Mediator{&json, JSONFormatName}, fail
	case EditorFormatName:
		logging.Debug("Using Editor outputer")
		editor, fail := NewEditor(config)
		return &Mediator{&editor, EditorFormatName}, fail
	case EditorV0FormatName:
		logging.Debug("Using EditorV0 outputer")
		editor0, fail := NewEditorV0(config)
		return &Mediator{&editor0, EditorV0FormatName}, fail
	}

	return nil, FailNotRecognized.New(locale.Tr("err_unknown_format", string(formatName)))
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
