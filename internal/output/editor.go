package output

import (
	"github.com/ActiveState/cli/internal/failures"
)

type Editor struct {
	JSON
}

// Type tells callers what type of outputer we are
func (f *Editor) Type() Format {
	return EditorFormatName
}

func NewEditor(config *Config) (Editor, error) {
	json, fail := NewJSON(config)
	if fail != nil {
		return Editor{}, fail
	}

	return Editor{json}, nil
}
