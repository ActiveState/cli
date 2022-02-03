package config

import (
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

type Set struct {
	out output.Outputer
	cfg *config.Instance
}

type SetParams struct {
	Key   Key
	Value string
}

func NewSet(prime primeable) *Set {
	return &Set{prime.Output(), prime.Config()}
}

func (s *Set) Run(params SetParams) error {
	err := s.cfg.Set(params.Key.String(), params.Value)
	if err != nil {
		return locale.WrapError(err, "err_cofing_set", fmt.Sprintf("Could not set value %s for key %s", params.Value, params.Key))
	}

	s.out.Print(locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", params.Key.String(), params.Value))
	return nil
}
