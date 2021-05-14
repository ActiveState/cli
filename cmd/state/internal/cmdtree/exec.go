package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/exec"
)

func newExecCommand(prime *primer.Values, args ...string) *captain.Command {
	runner := exec.New(prime)

	params := exec.NewParams()

	cmd := captain.NewCommand(
		"exec",
		"",
		locale.T("exec_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "path",
				Description: locale.Tl("flag_state_exec_path_description", "Path to project that is providing the default environment."),
				Value:       &params.Path,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params, args...)
		},
	)
	cmd.SetSkipChecks(true)
	cmd.SetDeferAnalytics(true)

	// Cobra will handle the `--` delimiter if flag parsing is enabled.
	// If the delimeter is not present we have to disable flag parsing
	// to ensure flags are passed to the command rather than
	// parsed as a flag for `state exec`
	if !strings.Contains(strings.Join(args, " "), " -- ") {
		cmd.SetDisableFlagParsing(true)
	}

	cmd.SetGroup(EnvironmentGroup)

	return cmd
}
