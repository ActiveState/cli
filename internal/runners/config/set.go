package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
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
	regex := regexp.MustCompile(`^[A-Za-z0-9\.]+$`)
	if !regex.MatchString(params.Key) {
		return locale.NewInputError("err_config_invalid_key", "The config can only consist of alphanumeric characters and a '.'")
	}

	var value interface{}
	value = params.Value
	if v, ok := keys[strings.ToLower(params.Key)]; ok {
		switch v {
		case "bool":
			value = cast.ToBool(value)
		case "int":
			value = cast.ToInt(value)
		}
	}

	err := s.cfg.Set(params.Key, value)
	if err != nil {
		return locale.WrapError(err, "err_cofing_set", fmt.Sprintf("Could not set value %s for key %s", value, params.Key))
	}

	s.out.Print(locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", params.Key, params.Value))
	return nil
}
