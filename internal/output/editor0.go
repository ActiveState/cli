package output

import (
	"github.com/ActiveState/cli/internal/failures"
)

type EditorV0 struct {
	JSON
}

// Error will marshal and print the given value to the error writer
// NOTE that EditorV0 always prints to the output writer, the error writer is unused.
func (f *EditorV0) Error(value interface{}) {
	f.JSON.Print(value)
}

// Type tells callers what type of outputer we are
func (f *EditorV0) Type() Format {
	return EditorV0FormatName
}

func NewEditorV0(config *Config) (EditorV0, error) {
	json, fail := NewJSON(config)
	json.printNUL = false
	if fail != nil {
		return EditorV0{}, fail
	}

	return EditorV0{json}, nil
}
