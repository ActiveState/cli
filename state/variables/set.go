package variables

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
)

func buildSetCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "set",
		Description: "variables_set_cmd_description",
		Run:         cmd.ExecuteSet,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "variables_set_arg_name_name",
				Description: "variables_set_arg_name_description",
				Variable:    &cmd.Args.Name,
				Required:    true,
			},
			&commands.Argument{
				Name:        "variables_set_arg_value_name",
				Description: "variables_set_arg_value_description",
				Variable:    &cmd.Args.Value,
				Required:    true,
			},
		},
	}
}

// ExecuteSet processes the `secrets set` command.
func (cmd *Command) ExecuteSet(_ *cobra.Command, args []string) {
	prj := project.Get()
	variable := prj.InitVariable(cmd.Args.Name)

	fail := variable.Save(cmd.Args.Value)
	if fail != nil {
		failures.Handle(fail, locale.T("variables_err"))
	}
}
