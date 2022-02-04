package config

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"

	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

type Get struct {
	out output.Outputer
	cfg *config.Instance
}

type GetParams struct {
	Key Key
}

func NewGet(prime primeable) *Get {
	return &Get{prime.Output(), prime.Config()}
}

func (g *Get) Run(params GetParams) error {
	value := g.cfg.Get(params.Key.String())
	if value == nil {
		return locale.NewInputError("err_config_not_found", "No config value for key: {{.V0}}", params.Key.String())
	}

	value, err := configMediator.GetRule(params.Key.String()).GetEvent(value)
	if err != nil {
		return locale.WrapError(err, "err_config_get_event", "Could not retrieve config value, if this continues to happen please contact support.")
	}

	g.out.Print(value)
	return nil
}
