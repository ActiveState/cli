package cmdtree

import (
	"github.com/ActiveState/cli/internal-as/captain"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/primer"
	"github.com/ActiveState/cli/internal/runners/prepare"
)

func newPrepareCommand(prime *primer.Values) *captain.Command {
	runner := prepare.New(prime)

	cmd := captain.NewCommand(
		"_prepare",
		"",
		locale.Tl("prepare_description", "Prepare environment for use with the State Tool."),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(c *captain.Command, _ []string) error {
			return runner.Run(c)
		},
	)

	cmd.SetHidden(true)

	return cmd
}

func newPrepareCompletionsCommand(prime *primer.Values) *captain.Command {
	runner := prepare.NewCompletions(prime)

	cmd := captain.NewCommand(
		"completions",
		"",
		"",
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(c *captain.Command, _ []string) error {
			return runner.Run(c)
		},
	)

	cmd.SetHidden(true)

	return cmd
}
