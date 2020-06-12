package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/packages"
)

func newPackagesCommand(out output.Outputer) *captain.Command {
	runner := packages.NewList(out)

	params := packages.ListRunParams{}

	cmd := captain.NewCommand(
		"packages",
		locale.T("package_cmd_description"),
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

func newPackagesAddCommand(out output.Outputer) *captain.Command {
	runner := packages.NewAdd(out)

	params := packages.AddRunParams{}

	return captain.NewCommand(
		"add",
		locale.T("package_add_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "package_arg_nameversion",
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

func newPackagesUpdateCommand(out output.Outputer) *captain.Command {
	runner := packages.NewUpdate(out)

	params := packages.UpdateRunParams{}

	return captain.NewCommand(
		"update",
		locale.T("package_update_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "package_arg_nameversion",
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

func newPackagesRemoveCommand(out output.Outputer) *captain.Command {
	runner := packages.NewRemove(out)

	params := packages.RemoveRunParams{}

	return captain.NewCommand(
		"remove",
		locale.T("package_remove_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "package_arg_name",
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

func newPackagesImportCommand(out output.Outputer) *captain.Command {
	runner := packages.NewImport(out)

	params := packages.NewImportRunParams()

	return captain.NewCommand(
		"import",
		locale.T("package_import_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "file",
				Description: locale.T("package_import_flag_filename_description"),
				Value:       &params.FileName,
			},
			{
				Name:        "force",
				Description: locale.T("package_import_flag_force_description"),
				Value:       &params.Force,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(*params)
		},
	)
}

func newPackagesSearchCommand(out output.Outputer) *captain.Command {
	runner := packages.NewSearch(out)

	params := packages.SearchRunParams{}

	return captain.NewCommand(
		"search",
		locale.T("package_search_cmd_description"),
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
				Name:        "package_arg_name",
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
