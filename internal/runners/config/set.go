package config

import (
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/spf13/cast"
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
	var value interface{}
	switch rules.Get(params.Key).allowedType {
	case Bool:
		value = cast.ToBool(value)
	case Int:
		value = cast.ToInt(value)
	default:
		value = params.Value
	}

	err := s.cfg.Set(params.Key.String(), value)
	if err != nil {
		return locale.WrapError(err, "err_cofing_set", fmt.Sprintf("Could not set value %s for key %s", value, params.Key))
	}

	err = rules.Get(params.Key).setEvent()
	if err != nil {
		logging.Error("Could not execute additional logic on config set")
	}

	s.out.Print(locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", params.Key.String(), params.Value))
	return nil
}
