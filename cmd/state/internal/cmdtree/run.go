package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/run"
)

func newRunCommand(prime *primer.Values) *captain.Command {
	runner := run.New(prime)

	var name string

	cmd := captain.NewCommand(
		"run",
		"",
		locale.T("run_description"),
		prime,
		nil,
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_run_name"),
				Description: locale.T("arg_state_run_name_description"),
				Value:       &name,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			if name == "-h" || name == "--help" {
				prime.Output().Print(ccmd.UsageText())
				return nil
			} else if name == "-v" || name == "--verbose" {
				if len(args) > 1 {
					name, args = args[1], args[1:]
				} else {
					name, args = "", []string{}
				}
			}

			if name != "" && len(args) > 0 {
				args = args[1:]
			}

			return runner.Run(name, args)
		},
	)

	cmd.SetGroup(EnvironmentGroup)
	cmd.SetDisableFlagParsing(true)
	cmd.SetHasVariableArguments()

	return cmd
}
