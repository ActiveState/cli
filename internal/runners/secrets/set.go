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
		return err
	}

	secret, fail := getSecret(s.proj, params.Name)
	if fail != nil {
		return locale.WrapError(fail, "secrets_err_values")
	}

	if fail = secret.Save(params.Value); fail != nil {
		return locale.WrapError(fail, "secrets_err_try_save", "Cannot save secret")
	}

	return nil
}
