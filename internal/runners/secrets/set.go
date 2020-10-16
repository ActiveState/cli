package secrets

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type setPrimeable interface {
	primer.Projecter
}

type SetRunParams struct {
	Name  string
	Value string
}

type Set struct {
	proj *project.Project
}

func NewSet(p setPrimeable) *Set {
	return &Set{
		proj: p.Project(),
	}
}

func (s *Set) Run(params SetRunParams) error {
	if err := checkSecretsAccess(s.proj); err != nil {
		return err
	}

	secret, fail := getSecret(params.Name)
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	if fail = secret.Save(params.Value); fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	return nil
}
