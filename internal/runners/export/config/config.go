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

type configMap map[string]string

func (f *configMap) MarshalOutput(format output.Format) interface{} {
	return f
}

func (f *configMap) MarshalStructured(format output.Format) interface{} {
	return f
}

type configValue struct {
	Value string `json:"value"`
}

func (f *configValue) MarshalOutput(format output.Format) interface{} {
	return f.Value
}

func (f *configValue) MarshalStructured(format output.Format) interface{} {
	return f
}

func New(prime primeable) *Config {
	return &Config{prime.Output(), prime.Config()}
}

func (c *Config) Run(cmd *captain.Command, params *ConfigParams) error {
	output := configMap{
		Dir.String(): c.cfg.ConfigPath(),
	}

	filteredOutput := output
	if params.Filter.terms != nil {
		filteredOutput = map[string]string{}
		for _, term := range params.Filter.terms {
			if value, ok := output[term.String()]; ok {
				filteredOutput[term.String()] = value
				if len(params.Filter.terms) == 1 {
					c.out.Print(&configValue{value})
					return nil
				}
			}
		}
	}

	c.out.Print(&filteredOutput)
	return nil
}
