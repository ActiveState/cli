package secrets

import (
	"strings"

	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/pkg/project"
)

func getSecret(proj *project.Project, namespace string, cfg keypairs.Configurable) (*project.Secret, error) {
	n := strings.Split(namespace, ".")
	if len(n) != 2 {
		return nil, locale.NewInputError("secrets_err_invalid_namespace", "", namespace)
	}

	secretScope, err := project.NewSecretScope(n[0])
	if err != nil {
		return nil, err
	}
	secretName := n[1]

	return proj.InitSecret(secretName, secretScope, cfg), nil
}

func getSecretWithValue(proj *project.Project, name string, cfg keypairs.Configurable) (*project.Secret, *string, error) {
	secret, err := getSecret(proj, name, cfg)
	if err != nil {
		return nil, nil, err
	}

	val, err := secret.ValueOrNil()
	if err != nil {
		return nil, nil, err
	}

	return secret, val, nil
}
