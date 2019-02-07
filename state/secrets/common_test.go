package secrets_test

import (
	"strings"

	"github.com/ActiveState/cli/pkg/projectfile"
	yaml "gopkg.in/yaml.v2"
)

func loadSecretsProject() (*projectfile.Project, error) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: SecretProject
owner: SecretOrg
secrets:
  - name: undefined-secret
  - name: org-secret
  - name: proj-secret
    project: true
  - name: user-secret
    user: true
  - name: user-proj-secret
    project: true
    user: true
  - name: org-secret-with-proj-value
  - name: proj-secret-with-user-value
    project: true
  - name: user-secret-with-user-proj-value
    user: true
  - name: proj-secret-only-org-available
    project: true
  - name: user-secret-only-proj-available
    user: true
  - name: user-proj-secret-only-user-available
    user: true
    project: true
  - name: bad-base64-encoded-secret
  - name: invalid-encryption-secret
scripts:
  - name: echo-org-secret
    value: echo ${secrets.org-secret}
  - name: echo-upper-org-secret
    value: echo ${secrets.ORG-SECRET}
`)

	return project, yaml.Unmarshal([]byte(contents), project)
}
