package output

import "github.com/ActiveState/cli/internal/errs"

type Editor struct {
	JSON
}

// Type tells callers what type of outputer we are
func (f *Editor) Type() Format {
	return EditorFormatName
}

func NewEditor(config *Config) (Editor, error) {
	json, err := NewJSON(config)
	if err != nil {
		return Editor{}, errs.Wrap(err, "NewJSON failed")
	}

	return Editor{json}, nil
}
