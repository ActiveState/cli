package variables

import (
	"fmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
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
				Variable:    &cmd.Args.SecretName,
				Required:    true,
			},
		},
	}
}

// ExecuteGet processes the `secrets get` command.
func (cmd *Command) ExecuteGet(_ *cobra.Command, args []string) {
	expanderFn := NewExpander(cmd.secretsClient)
	if value, failure := expanderFn(cmd.Args.SecretName, projectfile.Get()); failure != nil {
		failures.Handle(failure, locale.T("variables_err"))
	} else {
		fmt.Print(value) // we don't want a newline at the end
	}
}
