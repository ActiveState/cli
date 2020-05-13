package output

import (
	"github.com/ActiveState/cli/internal/failures"
)

type EditorV0 struct {
	JSON
}

// Type tells callers what type of outputer we are
func (f *EditorV0) Type() Format {
	return EditorV0FormatName
}

func NewEditorV0(config *Config) (EditorV0, *failures.Failure) {
	json, fail := NewJSON(config)
	if fail != nil {
		return EditorV0{}, fail
	}

	return EditorV0{json}, nil
}
