package variables_test

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
variables:
  - name: undefined-secret
    value: 
      pullfrom: organization
      share: organization
  - name: org-secret
    value: 
      pullfrom: organization
      share: organization
  - name: proj-secret
    value: 
      pullfrom: project
      share: organization
  - name: user-secret
    value: 
      pullfrom: organization
  - name: user-proj-secret
    value: 
      pullfrom: project
  - name: org-secret-with-proj-value
    value: 
      pullfrom: organization
      share: organization
  - name: proj-secret-with-user-value
    value: 
      pullfrom: project
  - name: user-secret-with-user-proj-value
    value: 
      pullfrom: organization
  - name: proj-secret-only-org-available
    value: 
      pullfrom: project
      share: organization
  - name: user-secret-only-proj-available
    value: 
      pullfrom: project
  - name: user-proj-secret-only-user-available
    value: 
      pullfrom: project
  - name: bad-base64-encoded-secret
    value: 
      pullfrom: organization
      share: organization
  - name: invalid-encryption-secret
    value: 
      pullfrom: organization
      share: organization
`)

	err := yaml.Unmarshal([]byte(contents), project)
	if err != nil {
		return nil, err
	}

	return project, project.Parse()
}
