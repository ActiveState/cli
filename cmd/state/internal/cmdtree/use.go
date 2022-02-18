package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/use"
	"github.com/ActiveState/cli/pkg/project"
)

const useCmdName = "use"

func newUseCommand(prime *primer.Values) *captain.Command {
	runner := use.NewUse(prime)

	params := &use.Params{
		Namespace: &project.NamespacedOptionalOwner{},
	}

	cmd := captain.NewCommand(
		useCmdName,
		"",
		"Switch to using the given project",
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_activate_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetWeight(100)
	cmd.SetGroup(EnvironmentGroup)
	return cmd
}
