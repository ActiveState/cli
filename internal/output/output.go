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
	logging.Debug("Requested outputer for %s", formatName)

	switch formatName {
	case "", PlainFormatName:
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

// Config is the thing we pass to Outputer constructors
type Config struct {
	OutWriter   io.Writer
	ErrWriter   io.Writer
	Colored     bool
	Interactive bool
}
