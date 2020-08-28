package secrets

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

func buildSetCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "set",
		Description: "secrets_set_cmd_description",
		Run:         cmd.ExecuteSet,

		Arguments: []*commands.Argument{
			{
				Name:        "secrets_set_arg_name",
				Description: "secrets_set_arg_name_description",
				Variable:    &cmd.Args.Name,
				Required:    true,
			},
			{
				Name:        "secrets_set_arg_value_name",
				Description: "secrets_set_arg_value_description",
				Variable:    &cmd.Args.Value,
				Required:    true,
			},
		},
	}
}

// ExecuteSet processes the `secrets set` command.
func (cmd *Command) ExecuteSet(_ *cobra.Command, _ []string) {
	secret, fail := getSecret(cmd.Args.Name)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err"))
		cmd.config.Exiter(1)
		return
	}

	fail = secret.Save(cmd.Args.Value)
	if fail != nil {
		failures.Handle(fail, locale.T("secrets_err"))
		cmd.config.Exiter(1)
		return
	}
}
