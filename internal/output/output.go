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
	JSONFormatName            = "json"      // plain json
	EditorFormatName          = "editor"    // alias of "json"
	EditorV0FormatName        = "editor.v0" // for Komodo: alias of "json"
)

// FailNotRecognized is a failure due to the format not being recognized
var FailNotRecognized = failures.Type("output.fail.not.recognized", failures.FailInput)

// Outputer is the initialized formatter
type Outputer interface {
	Print(value interface{})
	Error(value interface{})
	Notice(value interface{})
	Config() *Config
}

// New constructs a new Outputer according to the given format name
func New(formatName string, config *Config) (Outputer, *failures.Failure) {
	logging.Debug("Requested outputer for %s", formatName)

	format := Format(formatName)
	switch format {
	case "", PlainFormatName:
		logging.Debug("Using Plain outputer")
		plain, fail := NewPlain(config)
		return &mediator{&plain, PlainFormatName}, fail
	case JSONFormatName:
		logging.Debug("Using JSON outputer")
		json, fail := NewJSON(config)
		return &mediator{&json, JSONFormatName}, fail
	case EditorFormatName:
		logging.Debug("Using Editor outputer")
		editor, fail := NewEditor(config)
		return &mediator{&editor, EditorFormatName}, fail
	case EditorV0FormatName:
		logging.Debug("Using EditorV0 outputer")
		editor0, fail := NewEditorV0(config)
		return &mediator{&editor0, EditorV0FormatName}, fail
	}

	return nil, FailNotRecognized.New(locale.Tr("err_unknown_format", string(formatName)))
}

// Config is the thing we pass to Outputer constructors
type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
