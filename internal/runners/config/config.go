package config

import (
	"github.com/ActiveState/cli/internal/config"
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
