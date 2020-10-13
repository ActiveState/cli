package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/shim"
)

func newShimCommand(prime *primer.Values, args ...string) *captain.Command {
	runner := shim.New(prime)

	cmd := captain.NewCommand(
		"shim",
		locale.T("shim_title"),
		locale.T("shim_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(args...)
		},
	)
	cmd.SetSkipChecks(true)

	// Cobra will handle the `--` delimiter if flag parsing is enabled.
	// If the delimeter is not present we have to disable flag parsing
	// to ensure flags are passed to the shimmed command rather than
	// parsed as a flag for `state shim`
	if !strings.Contains(strings.Join(args, " "), " -- ") {
		cmd.SetDisableFlagParsing(true)
	}

	return cmd
}
