package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/project"
)

type configurable interface {
	project.ConfigAble
	ConfigPath() string
	CachePath() string
}

type Config struct {
	output  output.Outputer
	confirm confirmAble
	cfg     configurable
}

type ConfigParams struct {
	Force bool
}

func NewConfig(prime primeable) *Config {
	return newConfig(prime.Output(), prime.Prompt(), prime.Config())
}

func newConfig(out output.Outputer, confirm confirmAble, cfg configurable) *Config {
	return &Config{
		output:  out,
		confirm: confirm,
		cfg:     cfg,
	}
}

func (c *Config) Run(params *ConfigParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return locale.NewError("err_clean_cache_activated")
	}

	if !params.Force {
		ok, err := c.confirm.Confirm(locale.T("confirm"), locale.T("clean_config_confirm"), new(bool))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	logging.Debug("Removing config directory: %s", c.cfg.ConfigPath())
	return removeConfig(c.cfg)
}
