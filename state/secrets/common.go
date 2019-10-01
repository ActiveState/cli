package secrets

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/project"
)

func getSecret(namespace string) (*project.Secret, *failures.Failure) {
	n := strings.Split(namespace, ".")
	if len(n) != 2 {
		if len(n) != 1 {
			return nil, failures.FailUserInput.New("secrets_err_invalid_namespace")
		}

		// Default to project and warn that this method of retrieving secrets is deprecated
		print.Warning(locale.T("secrets_warn_deprecated_namespace"))
		n = []string{string(project.SecretScopeProject), n[0]}
	}

	secretScope, fail := project.NewSecretScope(n[0])
	if fail != nil {
		return nil, fail
	}
	secretName := n[1]

	return project.Get().InitSecret(secretName, secretScope), nil
}

func getSecretWithValue(name string) (*project.Secret, *string, *failures.Failure) {
	secret, fail := getSecret(name)
	if fail != nil {
		return nil, nil, fail
	}

	val, fail := secret.ValueOrNil()
	if fail != nil {
		return nil, nil, fail
	}

	return secret, val, nil
}
