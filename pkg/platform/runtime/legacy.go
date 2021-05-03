package runtime

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
)

// injectProjectDir replaces {{.ProjectDir}} with the current project in environment variables
// if projectDir is unspecified, the corresponding environment variables are extracted
// This is a dirty workaround until https://www.pivotaltracker.com/story/show/172033094 is implemented
func injectProjectDir(env map[string]string, projectDir string) map[string]string {
	templateMeta := struct {
		ProjectDir string
	}{projectDir}

	resultEnv := map[string]string{}
	for k, v := range env {
		// Dirty workaround until https://www.pivotaltracker.com/story/show/172033094 is implemented
		// This avoids projectDir dependant env vars from being written
		if projectDir == "" && strings.Contains(v, "ProjectDir") {
			continue
		}

		valueTemplate, err := template.New(k).Parse(v)
		if err != nil {
			logging.Error("Skipping env value with invalid value: %s:%s, error: %v", k, v, err)
			continue
		}
		var realValue bytes.Buffer
		err = valueTemplate.Execute(&realValue, templateMeta)
		if err != nil {
			logging.Error("Skipping env value whose value could not be parsed: %s:%s, error: %v", k, v, err)
			continue
		}
		resultEnv[k] = realValue.String()
	}
	return resultEnv
}
