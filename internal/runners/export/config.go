package export

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

// Filter is the --filter flag for the export config command, it implements captain.FlagMarshaler
type Filter int

const (
	Unset Filter = iota
	Dir
)

var lookup = map[Filter]string{
	Unset: "unset",
	Dir:   "dir",
}

func (f Filter) String() string {
	for k, v := range lookup {
		if k == f {
			return v
		}
	}
	return lookup[Unset]
}

func SupportedFilters() []string {
	var supported []string
	for k, v := range lookup {
		if k != Unset {
			supported = append(supported, v)
		}
	}

	return supported
}

func (f *Filter) Set(value string) error {
	for k, v := range lookup {
		if v == value && k != Unset {
			*f = k
			return nil
		}
	}

	return locale.NewError("err_invalid_filter", value, strings.Join(SupportedFilters(), ", "))
}

func (f Filter) Type() string {
	return "filter"
}

type Config struct {
	out output.Outputer
}

type ConfigParams struct {
	Filters []Filter
}

func NewConfig(prime primeable) *Config {
	return &Config{prime.Output()}
}

func (c *Config) Run(cmd *captain.Command, params ConfigParams) error {
	output := map[string]string{
		Dir.String(): config.ConfigPath(),
	}

	if params.Filters == nil {
		return c.printOutput(output)

	}

	if len(params.Filters) == 1 {
		c.out.Print(output[params.Filters[0].String()])
		return nil
	}

	filteredOutput := map[string]string{}
	for _, filter := range params.Filters {
		if value, ok := output[filter.String()]; ok {
			filteredOutput[filter.String()] = value
		}
	}

	return c.printOutput(output)
}

func (c *Config) printOutput(output map[string]string) error {
	data, err := json.Marshal(output)
	if err != nil {
		return locale.WrapError(err, "err_export_config_dir", "Could not marshal config data")
	}

	c.out.Print(string(data))
	return nil
}
