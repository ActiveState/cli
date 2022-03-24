package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/config"
)

func newConfigCommand(prime *primer.Values) *captain.Command {
	return captain.NewCommand(
		"config",
		locale.Tl("config_title", "Config"),
		locale.Tl("config_description", "Manage the State Tool configuration"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			runner, err := config.NewConfig(prime)
			if err != nil {
				return err
			}
			return runner.Run(ccmd.Usage)
		}).SetGroup(UtilsGroup)
}

func newConfigGetCommand(prime *primer.Values) *captain.Command {
	params := config.GetParams{}
	return captain.NewCommand(
		"get",
		locale.Tl("config_get_title", "Get config value"),
		locale.Tl("config_get_description", "Print config values to the terminal"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "key",
				Description: locale.Tl("arg_config_get_key", "Config key"),
				Required:    true,
				Value:       &params.Key,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			runner := config.NewGet(prime)
			return runner.Run(params)
		})
}

func newConfigSetCommand(prime *primer.Values) *captain.Command {
	params := config.SetParams{}
	return captain.NewCommand(
		"set",
		locale.Tl("config_set_title", "Set config value"),
		locale.Tl("config_set_description", "Set config values using the terminal"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "key",
				Description: locale.Tl("arg_config_set_key", "Config key"),
				Required:    true,
				Value:       &params.Key,
			},
			{
				Name:        "value",
				Description: locale.Tl("arg_config_set_value", "Config key"),
				Required:    true,
				Value:       &params.Value,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			runner := config.NewSet(prime)
			return runner.Run(params)
		})
}
