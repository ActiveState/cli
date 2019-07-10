package projectfile

import (
	"bytes"
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/alecthomas/template"
	"github.com/gobuffalo/packr"
)

var failTemplateLoad = failures.Type("projectfile.fail.templateload", failures.FailRuntime)

func loadTemplate(data map[string]interface{}, path string) *failures.Failure {
	box := packr.NewBox("../../assets/")
	tpl := box.String("activestate.yaml")
	t, err := template.New("activestateYAML").Parse(tpl)
	if err != nil {
		return failTemplateLoad.Wrap(err)
	}
	var out bytes.Buffer
	err = t.Execute(&out, data)
	if err != nil {
		return failures.FailTemplating.Wrap(err)
	}
	f, err := os.Create(path)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()

	_, err = f.Write([]byte(out.String()))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}
