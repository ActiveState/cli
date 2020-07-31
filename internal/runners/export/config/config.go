package config

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type Config struct {
	out output.Outputer
}

type ConfigParams struct {
	Filters Filters
}

type primeable interface {
	primer.Outputer
}

func New(prime primeable) *Config {
	return &Config{prime.Output()}
}

func (c *Config) Run(cmd *captain.Command, params *ConfigParams) error {
	output := map[string]string{
		Dir.String(): config.ConfigPath(),
	}

	filteredOutput := map[string]string{}
	if params.Filters.filters == nil {
		filteredOutput = output
	}

	for _, filter := range params.Filters.filters {
		if value, ok := output[filter.String()]; ok {
			filteredOutput[filter.String()] = value
			if len(params.Filters.filters) == 1 {
				c.out.Print(value)
				return nil
			}
		}
	}

	return c.printOutput(filteredOutput)
}

func (c *Config) printOutput(output map[string]string) error {
	data, err := json.Marshal(output)
	if err != nil {
		return locale.WrapError(err, "err_export_config_marshal", "Could not marshal config data")
	}

	c.out.Print(string(data))
	return nil
}
