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
	Unknown Filter = iota
	Dir
)

var lookup = map[Filter]string{
	Unknown: "unknown",
	Dir:     "dir",
}

func (f Filter) String() string {
	for k, v := range lookup {
		if k == f {
			return v
		}
	}
	return lookup[Unknown]
}

func supportedFilters() []string {
	var supported []string
	for k, v := range lookup {
		if k != Unknown {
			supported = append(supported, v)
		}
	}

	return supported
}

func SupportedFilters() string {
	return strings.Join(supportedFilters(), ", ")
}

func (f *Filter) Set(value string) error {
	for k, v := range lookup {
		if v == value && k != Unknown {
			*f = k
			return nil
		}
	}

	return locale.NewError("err_invalid_filter", value, SupportedFilters())
}

func (f Filter) Type() string {
	return "filter"
}

type Config struct {
	out output.Outputer
}

type ConfigParams struct {
	Filter Filter
}

type configOutput struct {
	Dir string `json:"dir"`
}

func NewConfig(prime primeable) *Config {
	return &Config{prime.Output()}
}

func (c *Config) Run(cmd *captain.Command, params ConfigParams) error {
	output := configOutput{config.ConfigPath()}

	if params.Filter == Dir {
		c.out.Print(output.Dir)
		return nil
	}

	data, err := json.Marshal(output)
	if err != nil {
		return locale.WrapError(err, "err_export_config_dir", "Could not marshal config data")
	}

	c.out.Print(string(data))
	return nil
}
