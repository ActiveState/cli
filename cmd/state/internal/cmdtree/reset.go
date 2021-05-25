package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/reset"
)

func newResetCommand(prime *primer.Values) *captain.Command {
	runner := reset.New(prime)

	return captain.NewCommand(
		"reset",
		locale.Tl("reset_title", "Restting commit"),
		locale.Tl("reset_description", "Reset to the most recent remote commit"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		},
	).SetGroup(VCSGroup)
}
