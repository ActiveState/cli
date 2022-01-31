package config

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

type Get struct {
	out output.Outputer
	cfg *config.Instance
}

type GetParams struct {
	Key string
}

func NewGet(prime primeable) *Get {
	return &Get{prime.Output(), prime.Config()}
}

func (g *Get) Run(params GetParams) error {
	err := validateKey(params.Key)
	if err != nil {
		return locale.WrapError(err, "err_config_invalid_key", "Invalid config key")
	}

	value := g.cfg.Get(params.Key)
	if value == nil {
		return locale.NewInputError("err_config_not_found", "No config value for key: {{.V0}}", params.Key)
	}

	err = getEvent(params.Key)
	if err != nil {
		logging.Error("Could not execute additional logic on config set")
	}

	g.out.Print(value)
	return nil
}

func getEvent(key string) error {
	value, ok := meta[key]
	if !ok {
		return nil
	}

	return value.getEvent()
}
