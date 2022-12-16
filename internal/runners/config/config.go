package config

import (
	"github.com/ActiveState/cli/internal-as/config"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/primer"
)

type Config struct {
	out output.Outputer
	cfg *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Configurer
	primer.SvcModeler
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
