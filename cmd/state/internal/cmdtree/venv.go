package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/venv"
	"github.com/ActiveState/cli/pkg/project"
)

const venvCmdName = "venv"

func newVenvCommand(prime *primer.Values) *captain.Command {
	runner := venv.NewVenv(prime)

	params := &venv.Params{
		Namespace: &project.NamespacedOptionalOwner{},
	}

	cmd := captain.NewCommand(
		venvCmdName,
		"",
		"Launch a new virtual environment for the given project",
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
	cmd.SetWeight(90)
	cmd.SetGroup(EnvironmentGroup)
	return cmd
}
