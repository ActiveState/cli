package secrets

import (
	"github.com/ActiveState/cli/internal/locale"
)

type SetRunParams struct {
	Name  string
	Value string
}

type Set struct {
}

func NewSet() *Set {
	return &Set{}
}

func (s *Set) Run(params SetRunParams) error {
	secret, fail := getSecret(params.Name)
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	if fail = secret.Save(params.Value); fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	return nil
}
