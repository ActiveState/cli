package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/initialize"
	"github.com/spf13/viper"
)

func newInitCommand() *captain.Command {
	initRunner := initialize.NewInit(viper.GetViper())

	var namespace, path, language string
	return captain.NewCommand(
		"init",
		[]*captain.Flag{
			{
				Name:        "language",
				Shorthand:   "",
				Description: locale.T("flag_state_init_language_description"),
				Type:        captain.TypeString,
				StringVar:   &language,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_init_namespace"),
				Description: locale.T("arg_state_init_namespace_description"),
				Variable:    &namespace,
			},
			&captain.Argument{
				Name:        locale.T("arg_state_init_path"),
				Description: locale.T("arg_state_init_path_description"),
				Variable:    &path,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return initRunner.Run(namespace, path, language)
		},
	)
}
