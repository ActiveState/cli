package output

import (
	"github.com/ActiveState/cli/internal/failures"
)

type EditorV0 struct {
	JSON
}

func NewEditorV0(config *Config) (EditorV0, *failures.Failure) {
	json, fail := NewJSON(config)
	if fail != nil {
		return EditorV0{}, fail
	}

	return EditorV0{json}, nil
}
