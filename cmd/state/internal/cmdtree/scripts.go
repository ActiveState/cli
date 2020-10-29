package cmdtree

import (
	"fmt"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/scripts"
)

func newScriptsCommand(prime *primer.Values) *captain.Command {
	runner := scripts.NewScripts(prime)

	inner := func(next captain.ExecuteFunc) captain.ExecuteFunc {
		return func(cmd *captain.Command, args []string) error {
			fmt.Println("scripts-only before")
			if err := next(cmd, args); err != nil {
				fmt.Println("err with scripts-only next")
				return err
			}
			fmt.Println("scripts-only after")
			return nil
		}
	}

	cmd := captain.NewCommand(
		"scripts",
		locale.Tl("scripts_title", "Listing Scripts"),
		locale.T("scripts_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run()
		},
	)

	cmd.SetGroup(AutomationGroup)
	cmd.SetInterceptChain(inner)

	return cmd
}

func newScriptsEditCommand(prime *primer.Values) *captain.Command {
	editRunner := scripts.NewEdit(prime)
	params := scripts.EditParams{}

	return captain.NewCommand(
		"edit",
		locale.Tl("scripts_edit_title", "Editing Script"),
		locale.T("edit_description"),
		prime.Output(),
		[]*captain.Flag{
			{
				Name:        "expand",
				Shorthand:   "e",
				Description: locale.T("edit_script_cmd_expand_flag"),
				Value:       &params.Expand,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("edit_script_cmd_name_arg"),
				Description: locale.T("edit_script_cmd_name_arg_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return editRunner.Run(&params)
		},
	)

}
