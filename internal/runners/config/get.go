package config

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
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

// TODO: How do we want to handle nested config values?
func (g *Get) Run(params GetParams) error {
	value := g.cfg.Get(params.Key)
	if value == nil {
		return locale.NewInputError("err_config_not_found", "No config value for key: {{.V0}}", params.Key)
	}

	g.out.Print(value)
	return nil
}
