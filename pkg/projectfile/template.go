package projectfile

import (
	"bytes"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
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

func sampleContent(goos, owner, project string) string {
	sampleKey := "sample_yaml"
	nixKey := "example_event_nix"
	winKey := "example_event_win"

	data := map[string]interface{}{
		"Owner":   owner,
		"Project": project,
	}

	content := locale.T(sampleKey, data)

	switch goos {
	case "linux", "darwin":
		content += locale.T(nixKey, data)
	case "windows":
		content += locale.T(winKey, data)
	default:
		content += locale.T(nixKey, data)
		content += locale.T(winKey, data)
	}

	return content
}
