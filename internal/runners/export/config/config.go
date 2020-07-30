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
	Filters []string
	filters []Filter
}

type primeable interface {
	primer.Outputer
}

func prepare(params *ConfigParams) error {
	for _, f := range params.Filters {
		filter := Unset

		err := filter.Set(f)
		if err != nil {
			return err
		}
		params.filters = append(params.filters, filter)
	}

	return nil
}

func NewConfig(prime primeable) *Config {
	return &Config{prime.Output()}
}

func (c *Config) Run(cmd *captain.Command, params *ConfigParams) error {
	err := prepare(params)
	if err != nil {
		return err
	}

	output := map[string]string{
		Dir.String(): config.ConfigPath(),
	}

	if params.Filters == nil {
		return c.printOutput(output)

	}

	if len(params.Filters) == 1 {
		c.out.Print(output[params.filters[0].String()])
		return nil
	}

	filteredOutput := map[string]string{}
	for _, filter := range params.filters {
		if value, ok := output[filter.String()]; ok {
			filteredOutput[filter.String()] = value
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
