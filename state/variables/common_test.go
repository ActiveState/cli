package variables_test

import (
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/pkg/projectfile"
)

func loadSecretsProject() (*projectfile.Project, error) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: SecretProject
owner: SecretOrg
variables:
  - name: undefined-secret
    value:
      store: organization
      share: organization
  - name: org-secret
    value:
      store: organization
      share: organization
  - name: proj-secret
    value:
      store: project
      share: organization
  - name: user-secret
    value:
      store: organization
  - name: user-proj-secret
    value:
      store: project
  - name: org-secret-with-proj-value
    value:
      store: organization
      share: organization
  - name: proj-secret-with-user-value
    value:
      store: project
      share: organization
  - name: user-secret-with-user-proj-value
    value:
      store: organization
  - name: proj-secret-only-org-available
    value:
      store: project
      share: organization
  - name: user-secret-only-proj-available
    value:
      store: organization
  - name: user-proj-secret-only-user-available
    value:
      store: project
  - name: bad-base64-encoded-secret
    value:
      store: organization
      share: organization
  - name: invalid-encryption-secret
    value:
      store: organization
      share: organization
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
