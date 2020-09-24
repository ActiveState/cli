package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/prepare"
)

func newPrepareCommand(prime *primer.Values) *captain.Command {
	runner := prepare.New(prime)

	cmd := captain.NewCommand(
		"_prepare",
		"for programmatic use only",
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)

	cmd.SetHidden(true)

	return cmd
}
