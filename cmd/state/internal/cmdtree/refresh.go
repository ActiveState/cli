package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/refresh"
	"github.com/ActiveState/cli/pkg/project"
)

func newRefreshCommand(prime *primer.Values) *captain.Command {
	runner := refresh.New(prime)

	params := &refresh.Params{
		Namespace: &project.Namespaced{AllowOmitOwner: true},
	}

	cmd := captain.NewCommand(
		"refresh",
		"",
		locale.Tl("refresh_description", "Updates the given project runtime based on its current configuration"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.Tl("arg_state_refresh_namespace_description", "The namespace of the project to update, or just the project name if previously used"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(EnvironmentUsageGroup)
	cmd.SetSupportsStructuredOutput()
	return cmd
}
