package config

import (
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
	Filter Filter
}

type primeable interface {
	primer.Outputer
}

func New(prime primeable) *Config {
	return &Config{prime.Output()}
}

func (c *Config) Run(cmd *captain.Command, params *ConfigParams) error {
	outputData := map[string]string{
		Dir.String(): config.ConfigPath(),
	}

	filteredOutput := outputData
	if params.Filter.terms != nil {
		filteredOutput = map[string]string{}
		for _, term := range params.Filter.terms {
			if value, ok := outputData[term.String()]; ok {
				filteredOutput[term.String()] = value
				if len(params.Filter.terms) == 1 {
					c.out.Print(value)
					return nil
				}
			}
		}
	}

	c.out.Notice(output.Title(locale.Tl("export_config_title", "Exporting Configuration Data")))

	c.out.Print(filteredOutput)
	return nil
}
