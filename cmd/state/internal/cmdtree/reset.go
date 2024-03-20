package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/reset"
)

func newResetCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	runner := reset.New(prime)
	params := &reset.Params{}

	return captain.NewCommand(
		"reset",
		locale.Tl("reset_title", "Reset to a Commit"),
		locale.Tl("reset_description", "Reset local checkout to a particular commit."),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("arg_state_reset_target", "target"),
				Description: locale.Tl("arg_state_reset_target_description", "The commit ID or branch name to reset to. If not specified, resets local checkout to be equal to the project on the platform"),
				Value:       &params.Target,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			params.Force = globals.NonInteractive
			return runner.Run(params)
		},
	).SetGroup(VCSGroup).SetSupportsStructuredOutput()
}
