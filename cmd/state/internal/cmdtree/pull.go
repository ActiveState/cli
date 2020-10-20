package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/pull"
)

func newPullCommand(prime *primer.Values) *captain.Command {
	runner := pull.New(prime)

	return captain.NewCommand(
		"pull",
		locale.Tl("pull_title", "Pulling Remote Project"),
		locale.Tl("pull_description", "Pull in the latest version of your project from the ActiveState Platform"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run()
		})
}
