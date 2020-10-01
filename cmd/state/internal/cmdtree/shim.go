package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/shim"
)

func newShimCommand(prime *primer.Values) *captain.Command {
	runner := shim.New(prime)

	cmd := captain.NewCommand(
		"shim",
		locale.T("shim_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}

			return runner.Run(args...)
		},
	)
	cmd.SetSkipChecks(true)
	cmd.SetDisableFlagParsing(true)

	return cmd
}
