package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/scripts"
)

func newScriptsCommand(prime *primer.Values) *captain.Command {
	runner := scripts.NewScripts(prime)

	return captain.NewCommand(
		"scripts",
		locale.Tl("scripts_title", "Listing Scripts"),
		locale.T("scripts_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		}).SetGroup(AutomationGroup).SetUnstable(true)
}

func newScriptsEditCommand(prime *primer.Values) *captain.Command {
	editRunner := scripts.NewEdit(prime)
	params := scripts.EditParams{}

	return captain.NewCommand(
		"edit",
		locale.Tl("scripts_edit_title", "Editing Script"),
		locale.T("edit_description"),
		prime,
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
