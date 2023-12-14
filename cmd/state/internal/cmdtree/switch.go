package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/swtch"
)

func newSwitchCommand(prime *primer.Values) *captain.Command {
	runner := swtch.New(prime)

	params := swtch.SwitchParams{}

	cmd := captain.NewCommand(
		"switch",
		locale.Tl("switch_title", "Switching"),
		locale.Tl("switch_description", "Switch to a branch, commit, or tag"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("switch_arg_identifier", "identifier"),
				Description: locale.Tl("switch_arg_identifier_description", "The commit or branch to switch to"),
				Value:       &params.Identifier,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		})

	cmd.SetGroup(EnvironmentSetupGroup)
	cmd.SetSupportsStructuredOutput()
	return cmd
}
