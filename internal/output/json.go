package output

import (
	"encoding/json"
	"fmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

const JSONFormatName = "json"
const EditorFormatName = "editor" // JSON and editor are currently the same thing

// JSON ..
type JSON struct {
	cfg *Config
}

// NewJSON ..
func NewJSON(config *Config) (JSON, *failures.Failure) {
	return JSON{config}, nil
}

func (f *JSON) Print(value interface{}) {
	b, err := json.Marshal(value)
	if err != nil {
		logging.Error("Could not marshal value, error: %v", err)
		f.Error(fmt.Sprintf(locale.T("err_could_not_marshal_print")))
		return
	}
	f.cfg.OutWriter.Write(b)
}

func (f *JSON) Error(value interface{}) {
	errStruct := struct{ Error interface{} }{value}
	b, err := json.Marshal(errStruct)
	if err != nil {
		logging.Error("Could not marshal value, error: %v", err)
		b = []byte(locale.T("err_could_not_marshal_print"))
	}
	f.cfg.ErrWriter.Write(b)
}
