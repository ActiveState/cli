package output

import "github.com/ActiveState/cli/internal/errs"

type EditorV0 struct {
	JSON
}

// Error will marshal and print the given value to the error writer
// NOTE that EditorV0 always prints to the output writer, the error writer is unused.
func (f *EditorV0) Error(value interface{}) {
	f.JSON.Print(toStructuredError(value))
}

// Type tells callers what type of outputer we are
func (f *EditorV0) Type() Format {
	return EditorV0FormatName
}

func NewEditorV0(config *Config) (EditorV0, error) {
	json, err := NewJSON(config)
	if err != nil {
		return EditorV0{}, errs.Wrap(err, "NewJSON failed")
	}

	return EditorV0{json}, nil
}
