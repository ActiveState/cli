package cmdtree

import (
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/initialize"
	"github.com/ActiveState/cli/pkg/project"
)

func newInitCommand() *captain.Command {
	initRunner := initialize.New(viper.GetViper())

	params := initialize.RunParams{
		Namespace: &project.Namespaced{},
	}

	return captain.NewCommand(
		"init",
		locale.T("init_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_init_path_description"),
				Value:       &params.Path,
			},
			{
				Name:      "skeleton",
				Shorthand: "",
				Description: locale.Tr(
					"flag_state_init_skeleton_description",
					initialize.RecognizedSkeletonStyles(),
				),
				Value: &params.Style,
			},
			{
				// Hidden flag for legacy Komodo support
				Name:   "language",
				Value:  &params.Language,
				Hidden: true,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_init_namespace"),
				Description: locale.T("arg_state_init_namespace_description"),
				Value:       params.Namespace,
				Required:    true,
			},
			{
				Name:        locale.T("arg_state_init_language"),
				Description: locale.T("arg_state_init_language_description"),
				Value:       &params.Language,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return initRunner.Run(&params)
		},
	)
}
