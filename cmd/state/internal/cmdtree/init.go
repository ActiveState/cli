package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/initialize"
	"github.com/ActiveState/cli/pkg/project"
)

func newInitCommand(prime *primer.Values) *captain.Command {
	initRunner := initialize.New(prime)

	params := initialize.RunParams{}

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
				Value:       &params.Namespace,
			},
			{
				Name:        locale.T("arg_state_init_path"),
				Description: locale.T("arg_state_init_path_description"),
				Value:       &params.Path,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			if params.Namespace != "" {
				ns, err := project.ParseNamespace(params.Namespace)
				if err != nil {
					// If the namespace was invalid but an argument was passed, we
					// assume it's a project name and not an owner.
					logging.Debug("Could not parse namespace: %v", err)
					params.ProjectName = params.Namespace
				} else {
					params.ParsedNS = ns
				}
			}
			return initRunner.Run(&params)
		},
	).SetGroup(EnvironmentSetupGroup).SetSupportsStructuredOutput()
}
