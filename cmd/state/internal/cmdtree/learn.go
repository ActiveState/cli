package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/learn"
)

func newLearnCommand(prime *primer.Values) *captain.Command {
	learnRunner := learn.New(prime)

	return captain.NewCommand(
		"learn",
		locale.Tl("learn_title", "Learn about the State Tool"),
		locale.Tl("learn_description", "Read the State Tool cheat sheet to learn about common commands"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return learnRunner.Run()
		}).SetGroup(UtilsGroup)
}
