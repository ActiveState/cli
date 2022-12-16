package config

import (
	"github.com/ActiveState/cli/internal-as/captain"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/primer"
)

type configurable interface {
	ConfigPath() string
}

type Config struct {
	out output.Outputer
	cfg configurable
}

type ConfigParams struct {
	Filter Filter
}

type primeable interface {
	primer.Outputer
	primer.Configurer
}

func New(prime primeable) *Config {
	return &Config{prime.Output(), prime.Config()}
}

func (c *Config) Run(cmd *captain.Command, params *ConfigParams) error {
	output := map[string]string{
		Dir.String(): c.cfg.ConfigPath(),
	}

	filteredOutput := output
	if params.Filter.terms != nil {
		filteredOutput = map[string]string{}
		for _, term := range params.Filter.terms {
			if value, ok := output[term.String()]; ok {
				filteredOutput[term.String()] = value
				if len(params.Filter.terms) == 1 {
					c.out.Print(value)
					return nil
				}
			}
		}
	}

	c.out.Print(filteredOutput)
	return nil
}
