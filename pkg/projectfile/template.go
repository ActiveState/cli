package projectfile

import (
	"bytes"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/alecthomas/template"
	"github.com/gobuffalo/packr"
)

var failTemplateLoad = failures.Type("projectfile.fail.templateload", failures.FailRuntime)

func loadTemplate(path string, data map[string]interface{}) (*bytes.Buffer, *failures.Failure) {
	box := packr.NewBox("../../assets/")
	tpl := box.String("activestate.yaml")

	t, err := template.New("activestateYAML").Parse(tpl)
	if err != nil {
		return nil, failTemplateLoad.Wrap(err)
	}

	var out bytes.Buffer
	err = t.Execute(&out, data)
	if err != nil {
		return nil, failures.FailTemplating.Wrap(err)
	}

	return &out, nil
}
