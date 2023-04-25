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

type mapOutput struct {
	map_ map[string]string
}

func (o *mapOutput) MarshalOutput(format output.Format) interface{} {
	return o.map_
}

func (o *mapOutput) MarshalStructured(format output.Format) interface{} {
	return o.map_
}

type valueOutput struct {
	Value string `json:"value"`
}

func (o *valueOutput) MarshalOutput(format output.Format) interface{} {
	return o.Value
}

func (o *valueOutput) MarshalStructured(format output.Format) interface{} {
	return o
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
					c.out.Print(&valueOutput{value})
					return nil
				}
			}
		}
	}

	c.out.Print(&mapOutput{filteredOutput})
	return nil
}
