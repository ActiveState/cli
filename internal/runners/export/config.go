package export

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
)

const (
	FilterDir = "dir"
)

func RecognizedFilters() string {
	filterLookup := []string{
		FilterDir,
	}
	return strings.Join(filterLookup, ",")
}

type Config struct {
	out output.Outputer
}

type ConfigParams struct {
	Filter string
}

func NewConfig(prime primeable) *Config {
	return &Config{prime.Output()}
}

func (c *Config) Run(cmd *captain.Command, params ConfigParams) error {
	if params.Filter == FilterDir {
		c.out.Print(config.ConfigPath())
		return nil
	}
	return cmd.Usage()

}

type Directory struct {
	out output.Outputer
}

type directoryOutput struct {
	Dir string `json:"dir"`
}

func NewDirectory(prime primeable) *Directory {
	return &Directory{prime.Output()}
}

func (d *Directory) Run() error {
	output := directoryOutput{config.ConfigPath()}

	data, err := json.Marshal(output)
	if err != nil {
		return locale.WrapError(err, "err_export_config_dir", "Could not marshal config data")
	}
	d.out.Print(string(data))
	return nil
}
