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
		locale.Tl("reset_title", "Reset to Same Commit as Platform Project"),
		locale.Tl("reset_description", "Reset local checkout to be equal to the project on the platform."),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		},
	).SetGroup(VCSGroup)
}
