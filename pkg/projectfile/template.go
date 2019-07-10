package projectfile

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/alecthomas/template"
	"github.com/gobuffalo/packr"
)

var failTemplateLoad = failures.Type("projectfile.fail.templateload", failures.FailRuntime)

func loadTemplate(data map[string]interface{}, path string) (*bytes.Buffer, *failures.Failure) {
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

func writeTemplate(path string, content *bytes.Buffer) *failures.Failure {
	f, err := os.Create(path)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()
	fmt.Println(path)
	_, err = f.Write([]byte(content.String()))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}
