package secrets

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/project"
)

func getSecret(proj *project.Project, namespace string) (*project.Secret, error) {
	n := strings.Split(namespace, ".")
	if len(n) != 2 {
		return nil, failures.FailUserInput.New("secrets_err_invalid_namespace", namespace)
	}

	secretScope, fail := project.NewSecretScope(n[0])
	if fail != nil {
		return nil, fail
	}
	secretName := n[1]

	return proj.InitSecret(secretName, secretScope), nil
}

func getSecretWithValue(proj *project.Project, name string) (*project.Secret, *string, error) {
	secret, fail := getSecret(proj, name)
	if fail != nil {
		return nil, nil, fail
	}

	val, fail := secret.ValueOrNil()
	if fail != nil {
		return nil, nil, fail
	}

	return secret, val, nil
}
