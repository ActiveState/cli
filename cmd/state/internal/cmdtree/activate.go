package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/activate"
	"github.com/ActiveState/cli/pkg/project"
)

func newActivateCommand(prime *primer.Values) *captain.Command {
	runner := activate.NewActivate(prime)

	params := activate.ActivateParams{
		Namespace: &project.Namespaced{},
	}

	cmd := captain.NewCommand(
		"activate",
		locale.T("activate_project"),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("flag_state_activate_path_description"),
				Value:       &params.PreferredPath,
			},
			{
				Name:        "command",
				Shorthand:   "c",
				Description: locale.Tl("flag_state_activate_cmd_description", "Run given command in the activated shell"),
				Value:       &params.Command,
			},
			{
				Name:        "default",
				Description: locale.Tl("flag_state_activate_default_description", "Configures the project to be the global default project"),
				Value:       &params.Default,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(&params)
		},
	)
	cmd.SetDeferAnalytics(true)
	return cmd
}
