package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/run"
)

func newRunCommand() *captain.Command {
	runRunner := run.New()

	var name string
	return captain.NewCommand(
		"run",
		locale.T("run_description"),
		nil,
		[]*captain.Argument{
			{
				Name:        locale.T("arg_state_run_name"),
				Description: locale.T("arg_state_run_name_description"),
				Variable:    &name,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			if name != "" && len(args) > 0 {
				args = args[1:]
			}

			return runRunner.Run(name, args)
		},
	)
}
