package config

import (
	"regexp"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type Config struct {
	out output.Outputer
	cfg *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Configurer
}

func NewConfig(prime primeable) (*Config, error) {
	return &Config{
		out: prime.Output(),
		cfg: prime.Config(),
	}, nil
}

func (c *Config) Run(usageFunc func() error) error {
	return usageFunc()
}

func validateKey(key string) error {
	regex := regexp.MustCompile(`^[A-Za-z0-9\.]+$`)
	if !regex.MatchString(key) {
		return locale.NewInputError("err_config_invalid_key", "The config can only consist of alphanumeric characters and a '.'")
	}
	return nil
}
