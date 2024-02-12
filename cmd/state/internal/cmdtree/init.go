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
		"",
		locale.T("init_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "language",
				Description: locale.T("flag_state_init_language_description"),
				Value:       &params.Language,
			},
			{
				Name:        "private",
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
				Name:        locale.T("arg_state_init_path"),
				Description: locale.T("arg_state_init_path_description"),
				Value:       &params.Path,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return initRunner.Run(&params)
		},
	).SetGroup(EnvironmentSetupGroup).SetSupportsStructuredOutput()
}
