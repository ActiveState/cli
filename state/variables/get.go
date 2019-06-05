package variables

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
)

func buildGetCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "get",
		Description: "variables_get_cmd_description",
		Run:         cmd.ExecuteGet,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "variables_get_arg_name_name",
				Description: "variables_get_arg_name_description",
				Variable:    &cmd.Args.Name,
				Required:    true,
			},
		},
	}
}

// ExecuteGet processes the `secrets get` command.
func (cmd *Command) ExecuteGet(_ *cobra.Command, args []string) {
	prj := project.Get()
	variable := prj.InitVariable(cmd.Args.Name)
	value, fail := variable.Value()
	if fail != nil {
		failures.Handle(fail, locale.T("variables_err"))
	}

	print.Line(value)
}
