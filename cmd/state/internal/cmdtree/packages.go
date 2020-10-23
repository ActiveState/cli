package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/packages"
)

func newPackagesCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewList(prime)

	params := packages.ListRunParams{}

	cmd := captain.NewCommand(
		"packages",
		locale.Tl("package_title", "Listing Packages"),
		locale.T("package_cmd_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{
			{
				Name:        "commit",
				Description: locale.T("package_list_flag_commit_description"),
				Value:       &params.Commit,
			},
			{
				Name:        "package",
				Description: locale.T("package_list_flag_name_description"),
				Value:       &params.Name,
			},
			{
				Name:        "namespace",
				Description: locale.T("namespace_list_flag_project_description"),
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

func newPackagesAddCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewAdd(prime)

	params := packages.AddRunParams{}

	return captain.NewCommand(
		"add",
		locale.Tl("package_add_title", "Adding Package"),
		locale.T("package_add_cmd_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_nameversion"),
				Description: locale.T("package_arg_nameversion_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPackagesUpdateCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewUpdate(prime)

	params := packages.UpdateRunParams{}

	return captain.NewCommand(
		"update",
		locale.Tl("package_update_title", "Updating Packages"),
		locale.T("package_update_cmd_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_nameversion"),
				Description: locale.T("package_arg_nameversion_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPackagesRemoveCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewRemove(prime)

	params := packages.RemoveRunParams{}

	return captain.NewCommand(
		"remove",
		locale.Tl("package_remove_title", "Removing Package"),
		locale.T("package_remove_cmd_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_name"),
				Description: locale.T("package_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}

func newPackagesImportCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewImport(prime)

	params := packages.NewImportRunParams()

	return captain.NewCommand(
		"import",
		locale.Tl("package_import_title", "Importing Packages"),
		locale.T("package_import_cmd_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{
			{
				Name:        "force",
				Description: locale.T("package_import_flag_force_description"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("import_file", "File"),
				Description: locale.T("package_import_flag_filename_description"),
				Value:       &params.FileName,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(*params)
		},
	)
}

func newPackagesSearchCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewSearch(prime)

	params := packages.SearchRunParams{}

	return captain.NewCommand(
		"search",
		locale.Tl("package_search_title", "Searching Packages"),
		locale.T("package_search_cmd_description"),
		prime.Output(),
		[]captain.CommandGroup{},
		[]*captain.Flag{
			{
				Name:        "language",
				Description: locale.T("package_search_flag_language_description"),
				Value:       &params.Language,
			},
			{
				Name:        "exact-term",
				Description: locale.T("package_search_flag_exact-term_description"),
				Value:       &params.ExactTerm,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_name"),
				Description: locale.T("package_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
}
