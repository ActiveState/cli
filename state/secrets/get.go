package secrets

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

func buildGetCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "get",
		Description: "secrets_get_cmd_description",
		Run:         cmd.ExecuteGet,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "secrets_get_arg_name",
				Description: "secrets_get_arg_name_description",
				Variable:    &cmd.Args.Name,
				Required:    true,
			},
		},
	}
}

// ExecuteGet processes the `secrets get` command.
func (cmd *Command) ExecuteGet(_ *cobra.Command, args []string) {
	secret, value, fail := getSecretWithValue(cmd.Args.Name)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err"))
		return
	}

	if value == nil {
		err := "secrets_err_project_not_defined"
		if secret.IsUser() {
			err = "secrets_err_user_not_defined"
		}
		print.Error(locale.Tr(err, cmd.Args.Name))
		cmd.config.Exiter(1)
		return
	}

	print.Line(*value)
}
