package config

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Set struct {
	out       output.Outputer
	cfg       *config.Instance
	svcModel  *model.SvcModel
	analytics analytics.Dispatcher
}

type SetParams struct {
	Key   Key
	Value string
}

func NewSet(prime primeable) *Set {
	return &Set{prime.Output(), prime.Config(), prime.SvcModel(), prime.Analytics()}
}

func (s *Set) Run(params SetParams) error {
	// Cast to option type if applicable
	var value interface{}
	option := configMediator.GetOption(params.Key.String())
	if !configMediator.KnownOption(option) {
		return locale.NewInputError("unknown_config_key", "Unknown config key: {{.V0}}", params.Key.String())
	}
	switch option.Type {
	case configMediator.Bool:
		var err error
		value, err = strconv.ParseBool(params.Value)
		if err != nil {
			return locale.WrapInputError(err, "Invalid boolean value")
		}
	case configMediator.Int:
		var err error
		value, err = strconv.ParseInt(params.Value, 0, 0)
		if err != nil {
			return locale.WrapInputError(err, "Invalid integer value")
		}
	default:
		value = params.Value
	}

	value, err := option.SetEvent(value)
	if err != nil {
		return locale.WrapError(err, "err_config_set_event", "Could not store config value. If this continues to happen please contact support.")
	}

	key := params.Key.String()

	err = s.cfg.Set(key, value)
	if err != nil {
		return locale.WrapError(err, "err_config_set", fmt.Sprintf("Could not set value %s for key %s", params.Value, key))
	}

	// Notify listeners that this key has changed.
	configMediator.NotifyListeners(key)

	// Notify state-svc that this key has changed.
	if s.svcModel != nil {
		if err := s.svcModel.ConfigChanged(context.Background(), key); err != nil {
			logging.Error("Failed to report config change via state-svc: %s", errs.JoinMessage(err))
		}
	}
	s.sendEvent(key, params.Value, option)

	s.out.Print(output.Prepare(
		locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", key, params.Value),
		&struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
		}{
			key,
			params.Value,
		},
	))
	return nil
}

func (s *Set) sendEvent(key string, value string, option configMediator.Option) {
	action := constants.ActConfigSet
	if option.Type == configMediator.Bool {
		v, err := strconv.ParseBool(value)
		if err != nil {
			logging.Error("Could not parse bool value: %s", err)
			return
		}

		if !v {
			action = constants.ActConfigUnset
		}
	}

	s.analytics.EventWithLabel(constants.CatConfig, action, key)
}
