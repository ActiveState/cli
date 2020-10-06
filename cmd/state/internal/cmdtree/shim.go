package cmdtree

import (
	"strings"

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
			return runner.Run(args...)
		},
	)
	cmd.SetSkipChecks(true)

	if !strings.Contains(strings.Join(prime.Args(), " "), " -- ") {
		cmd.SetDisableFlagParsing(true)
	}
	return cmd
}
