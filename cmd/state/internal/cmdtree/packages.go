package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/packages"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func newPackagesCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewList(prime)

	params := packages.ListRunParams{}

	cmd := captain.NewCommand(
		"packages",
		locale.Tl("package_title", "Listing Packages"),
		locale.T("package_cmd_description"),
		prime,
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
			return runner.Run(params, model.NamespacePackage)
		},
	)

	cmd.SetGroup(PackagesGroup)
	cmd.SetAliases("pkg", "package")
	cmd.SetSupportsStructuredOutput()

	return cmd
}

func newInstallCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewInstall(prime)

	params := packages.InstallRunParams{}

	return captain.NewCommand(
		"install",
		locale.Tl("package_install_title", "Installing Package"),
		locale.T("package_install_cmd_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "ts",
				Description: locale.T("package_flag_ts_description"),
				Value:       &params.Timestamp,
			},
			{
				Name:        "revision",
				Shorthand:   "r",
				Description: locale.T("package_flag_rev_description"),
				Value:       &params.Revision,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_nameversion"),
				Description: locale.T("package_arg_nameversion_description"),
				Value:       &params.Package,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespacePackage)
		},
	).SetGroup(PackagesGroup).SetSupportsStructuredOutput()
}

func newUninstallCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewUninstall(prime)

	params := packages.UninstallRunParams{}

	return captain.NewCommand(
		"uninstall",
		locale.Tl("package_uninstall_title", "Uninstalling Package"),
		locale.T("package_uninstall_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_name"),
				Description: locale.T("package_arg_name_description"),
				Value:       &params.Package,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespacePackage)
		},
	).SetGroup(PackagesGroup).SetSupportsStructuredOutput()
}

func newImportCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	runner := packages.NewImport(prime)

	params := packages.NewImportRunParams()

	return captain.NewCommand(
		"import",
		locale.Tl("package_import_title", "Importing Packages"),
		locale.T("package_import_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("import_file", "File"),
				Description: locale.T("package_import_flag_filename_description"),
				Value:       &params.FileName,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			params.NonInteractive = globals.NonInteractive
			return runner.Run(params)
		},
	).SetGroup(PackagesGroup).SetSupportsStructuredOutput()
}

func newSearchCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewSearch(prime)

	params := packages.SearchRunParams{}

	return captain.NewCommand(
		"search",
		"",
		locale.T("package_search_cmd_description"),
		prime,
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
			{
				Name:        "ts",
				Description: locale.T("package_flag_ts_description"),
				Value:       &params.Timestamp,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_name"),
				Description: locale.T("package_arg_name_description"),
				Value:       &params.Ingredient,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespacePackage)
		},
	).SetGroup(PackagesGroup).SetSupportsStructuredOutput().SetUnstable(true)
}

func newInfoCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewInfo(prime)

	params := packages.InfoRunParams{}

	return captain.NewCommand(
		"info",
		"",
		locale.T("package_info_cmd_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "language",
				Description: locale.T("package_info_flag_language_description"),
				Value:       &params.Language,
			},
			{
				Name:        "ts",
				Description: locale.T("package_flag_ts_description"),
				Value:       &params.Timestamp,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_nameversion"),
				Description: locale.T("package_arg_nameversion_description"),
				Value:       &params.Package,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespacePackage)
		},
	).SetGroup(PackagesGroup).SetSupportsStructuredOutput()
}
