package config

import (
	"context"
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/spf13/cast"
)

type Set struct {
	out      output.Outputer
	cfg      *config.Instance
	svcModel *model.SvcModel
}

type SetParams struct {
	Key   Key
	Value string
}

func NewSet(prime primeable) *Set {
	return &Set{prime.Output(), prime.Config(), prime.SvcModel()}
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
		value = cast.ToBool(params.Value)
	case configMediator.Int:
		value = cast.ToInt(params.Value)
	default:
		value = params.Value
	}

	value, err := option.SetEvent(value)
	if err != nil {
		return locale.WrapError(err, "err_config_set_event", "Could not store config value, if this continues to happen please contact support.")
	}

	key := params.Key.String()

	err = s.cfg.Set(key, value)
	if err != nil {
		return locale.WrapError(err, "err_config_set", fmt.Sprintf("Could not set value %s for key %s", params.Value, params.Key))
	}

	// Notify listeners that this key has changed.
	configMediator.NotifyListeners(key)

	// Notify state-svc that this key has changed.
	if s.svcModel != nil {
		if err := s.svcModel.ConfigChanged(context.Background(), key); err != nil {
			logging.Debug("Failed to report config change via state-svc: %s", errs.JoinMessage(err))
		}
	}

	s.out.Print(locale.Tl("config_set_success", "Successfully set config key: {{.V0}} to {{.V1}}", params.Key.String(), params.Value))
	return nil
}
