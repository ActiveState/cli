package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/scripts"
	"github.com/ActiveState/cli/pkg/project"
)

func newScriptsCommand(pj *project.Project, output output.Outputer) *captain.Command {
	runner := scripts.NewScripts(pj, output)

	return captain.NewCommand(
		"scripts",
		locale.T("scripts_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		})
}

func newScriptsEditCommand(pj *project.Project, output output.Outputer) *captain.Command {
	editRunner := scripts.NewEdit(pj, output)
	params := scripts.EditParams{}

	return captain.NewCommand(
		"edit",
		locale.T("edit_description"),
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
