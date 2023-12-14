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
		locale.Tl("shell_description", "Starts a shell/prompt in a virtual environment for the given project runtime"),
		prime,
		[]*captain.Flag{
			{
				Name:        "cd",
				Shorthand:   "",
				Description: locale.Tl("flag_state_shell_cd_description", "Change to the project directory after starting virtual environment shell/prompt"),
				Value:       &params.ChangeDirectory,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_activate_namespace"),
				Description: locale.T("arg_state_shell_namespace_description"),
				Value:       params.Namespace,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(EnvironmentUsageGroup)
	cmd.SetAliases("prompt")
	return cmd
}
