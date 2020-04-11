package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/packages"
)

func newPackagesCommand() *captain.Command {
	runner := packages.NewList()

	params := packages.ListRunParams{}

	cmd := captain.NewCommand(
		"packages",
		locale.T("packages_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "commit",
				Description: "package_list_flag_commit_description",
				Value:       &params.Commit,
			},
			{
				Name:        "package",
				Description: "package_list_flag_name_description",
				Value:       &params.Name,
			},
			{
				Name:        "namespace",
				Description: "namespace_list_flag_project_description",
				Value:       &params.Project,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetAliases("pkg", "package")

	return cmd
}

func newPackagesAddCommand() *captain.Command {
	runner := packages.NewAdd()

	params := packages.AddRunParams{}

	return captain.NewCommand(
		"add",
		locale.T("packages_add_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "packages_arg_nameversion",
				Description: "packages_arg_nameversion_description",
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPackagesUpdateCommand() *captain.Command {
	runner := packages.NewUpdate()

	params := packages.UpdateRunParams{}

	return captain.NewCommand(
		"update",
		locale.T("packages_update_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "packages_arg_nameversion",
				Description: "packages_arg_nameversion_description",
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPackagesRemoveCommand() *captain.Command {
	runner := packages.NewRemove()

	params := packages.RemoveRunParams{}

	return captain.NewCommand(
		"remove",
		locale.T("packages_remove_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "packages_arg_name",
				Description: "packages_arg_name_description",
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}
