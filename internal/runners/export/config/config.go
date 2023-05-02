package config

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
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

type valueOutput struct {
	Value string `json:"value"`
}

func (c *Config) Run(cmd *captain.Command, params *ConfigParams) error {
	configOutput := map[string]string{
		Dir.String(): c.cfg.ConfigPath(),
	}

	filteredOutput := configOutput
	if params.Filter.terms != nil {
		filteredOutput = map[string]string{}
		for _, term := range params.Filter.terms {
			if value, ok := configOutput[term.String()]; ok {
				filteredOutput[term.String()] = value
				if len(params.Filter.terms) == 1 {
					c.out.Print(output.Prepare(value, &valueOutput{value}))
					return nil
				}
			}
		}
	}

	c.out.Print(output.Prepare(filteredOutput, filteredOutput))
	return nil
}
