package config

import (
	"fmt"
	"strings"

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
	Key   string
	Value string
}

func NewSet(prime primeable) *Set {
	return &Set{prime.Output(), prime.Config()}
}

func (s *Set) Run(params SetParams) error {
	err := validateKey(params.Key)
	if err != nil {
		return locale.WrapError(err, "err_config_invalid_key", "Invalid config key")
	}

	var value interface{}
	value = params.Value
	if v, ok := meta[strings.ToLower(params.Key)]; ok {
		switch v.Type {
		case Bool:
			value = cast.ToBool(value)
		case Int:
			value = cast.ToInt(value)
		}
	}

	err = s.cfg.Set(params.Key, value)
	if err != nil {
		return locale.WrapError(err, "err_cofing_set", fmt.Sprintf("Could not set value %s for key %s", value, params.Key))
	}

	err = setEvent(params.Key)
	if err != nil {
		logging.Error("Could not execute additional logic on config set")
	}

	s.out.Print(locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", params.Key, params.Value))
	return nil
}

func setEvent(key string) error {
	value, ok := meta[key]
	if !ok {
		return nil
	}

	return value.setEvent()
}
