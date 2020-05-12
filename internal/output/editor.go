package output

import (
	"github.com/ActiveState/cli/internal/failures"
)

type Editor struct {
	JSON
}

func NewEditor(config *Config) (Editor, *failures.Failure) {
	json, fail := NewJSON(config)
	if fail != nil {
		return Editor{}, fail
	}

	return Editor{json}, nil
}
