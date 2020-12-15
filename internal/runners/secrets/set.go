package secrets

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type setPrimeable interface {
	primer.Projecter
}

// SetRunParams tracks the info required for running Set.
type SetRunParams struct {
	Name  string
	Value string
}

// Set manages the setting execution context.
type Set struct {
	proj *project.Project
}

// NewSet prepares a set execution context for use.
func NewSet(p setPrimeable) *Set {
	return &Set{
		proj: p.Project(),
	}
}

// Run executes the set behavior.
func (s *Set) Run(params SetRunParams) error {
	if err := checkSecretsAccess(s.proj); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	secret, err := getSecret(s.proj, params.Name)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	if err = secret.Save(params.Value); err != nil {
		return locale.WrapError(err, "secrets_err_try_save", "Cannot save secret")
	}

	return nil
}
