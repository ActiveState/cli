package fork

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type resultEditorV0 struct {
	Result map[string]string    `json:"result,omitempty"`
	Error  *resultEditorV0Error `json:"error,omitempty"`
}

type resultEditorV0Error struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

func (o *forkOutput) editorV0Format() interface{} {
	return resultEditorV0{
		map[string]string{
			"OriginalOwner": o.source.Owner,
			"OriginalName":  o.source.Project,
			"NewOwner":      o.target.Owner,
			"NewName":       o.target.Project,
		},
		nil,
	}
}

type editorV0Error struct {
	parent error
}

func (e *editorV0Error) Error() string {
	return "editorV0Error wrapper"
}

func (e *editorV0Error) Unwrap() error {
	return e.parent
}

func (e *editorV0Error) AddTips(...string) {
	return
}

func (e *editorV0Error) ErrorTips() []string {
	return []string{}
}

func (e *editorV0Error) MarshalStructured(output.Format) interface{} {
	logging.Debug("Marshalling editorv0 error")
	var code int32 = 1
	for _, errInspect := range errs.Unpack(e.parent) {
		err, ok := errInspect.(error)
		if ok && errs.Matches(err, &model.ErrProjectNameConflict{}) {
			code = -16
		}
	}
	result := resultEditorV0{
		nil,
		&resultEditorV0Error{
			code,
			e.parent.Error(),
			"",
		},
	}
	return result
}