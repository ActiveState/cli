package variables_test

import (
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/pkg/projectfile"
)

func loadSecretsProject() (*projectfile.Project, error) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/SecretOrg/SecretProject/d7ebc72"
scripts:
  - name: echo-org-secret
    value: echo ${secrets.org-secret}
  - name: echo-upper-org-secret
    value: echo ${secrets.ORG-SECRET}
`)

	err := yaml.Unmarshal([]byte(contents), project)
	if err != nil {
		return nil, err
	}

	return project, project.Parse()
}
