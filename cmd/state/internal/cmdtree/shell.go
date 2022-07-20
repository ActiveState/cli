package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/shell"
	"github.com/ActiveState/cli/pkg/project"
)

const shellCmdName = "shell"

func newShellCommand(prime *primer.Values) *captain.Command {
	runner := shell.New(prime)

	params := &shell.Params{
		Namespace: &project.Namespaced{AllowOmitOwner: true},
	}

	cmd := captain.NewCommand(
		shellCmdName,
		"",
		locale.Tl("shell_description", "Starts a shell/prompt for the given project runtime"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.Tl("arg_state_shell_namespace_description", "The namespace of the project you wish to start a shell/prompt for"),
				Required:    true,
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(EnvironmentGroup)
	cmd.SetAliases("prompt")
	return cmd
}
