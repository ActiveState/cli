package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/initialize"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/viper"
)

func newInitCommand() *captain.Command {
	initRunner := initialize.New(viper.GetViper())

	params := initialize.RunParams{
		Namespace: &project.Namespace{},
	}

	return captain.NewCommand(
		"init",
		locale.T("init_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("arg_state_init_path_description"),
				Value:       &params.Path,
			},
			{
				Name:        "language",
				Shorthand:   "",
				Description: locale.T("flag_state_init_language_description"),
				Value:       &params.Language,
			},
			{
				Name:        "skeleton",
				Shorthand:   "",
				Description: locale.T("flag_state_init_skeleton_description"),
				Value:       &params.Style,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_init_namespace"),
				Description: locale.T("arg_state_init_namespace_description"),
				Value:       params.Namespace,
				Required:    true,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return initRunner.Run(&params)
		},
	)
}
