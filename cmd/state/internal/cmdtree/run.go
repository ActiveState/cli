package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/run"
)

func newRunCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	runner := run.New(prime)

	params := run.Params{}

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
				Value:       &params.ScriptName,
				Required:    true,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			if params.ScriptName == "-h" || params.ScriptName == "--help" {
				prime.Output().Print(ccmd.UsageText())
				return nil
			} else if params.ScriptName == "-v" || params.ScriptName == "--verbose" {
				if len(args) > 1 {
					params.ScriptName, args = args[1], args[1:]
				} else {
					params.ScriptName, args = "", []string{}
				}
			}

			if params.ScriptName != "" && len(args) > 0 {
				args = args[1:]
			}
			params.Args = args
			params.NonInteractive = globals.NonInteractive
			return runner.Run(params)
		},
	)

	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetDisableFlagParsing(true)
	cmd.SetHasVariableArguments()

	return cmd
}
