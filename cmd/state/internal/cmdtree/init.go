package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/initialize"
	"github.com/ActiveState/cli/pkg/project"
)

func newInitCommand(prime *primer.Values) *captain.Command {
	initRunner := initialize.New(prime)

	params := initialize.RunParams{
		Namespace: &project.Namespaced{},
	}

	return captain.NewCommand(
		"init",
		locale.Tl("init_title", "Initializing Project"),
		locale.T("init_description"),
		prime,
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
			{
				Name:        "private",
				Shorthand:   "",
				Description: locale.T("flag_state_init_private_flag_description"),
				Value:       &params.Private,
			},
		},
		[]*captain.Argument{
			{
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
	).SetGroup(EnvironmentGroup)
}
