package config

import (
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
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
	// Cast to rule type if applicable
	var value interface{}
	rule := configMediator.GetRule(params.Key.String())
	if !configMediator.KnownRule(rule) {
		return locale.NewInputError("unknown_config_key", "Unknown config key: {{.V0}}", params.Key.String())
	}
	switch rule.Type {
	case configMediator.Bool:
		value = cast.ToBool(params.Value)
	case configMediator.Int:
		value = cast.ToInt(params.Value)
	default:
		value = params.Value
	}

	value, err := rule.SetEvent(value)
	if err != nil {
		return locale.WrapError(err, "err_config_set_event", "Could not store config value, if this continues to happen please contact support.")
	}

	err = s.cfg.Set(params.Key.String(), value)
	if err != nil {
		return locale.WrapError(err, "err_config_set", fmt.Sprintf("Could not set value %s for key %s", params.Value, params.Key))
	}

	s.out.Print(locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", params.Key.String(), params.Value))
	return nil
}
