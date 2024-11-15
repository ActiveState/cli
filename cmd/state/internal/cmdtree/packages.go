package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/install"
	"github.com/ActiveState/cli/internal/runners/packages"
	"github.com/ActiveState/cli/internal/runners/uninstall"
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
	runner := install.New(prime, model.NamespacePackage)

	params := install.Params{}
	force := false

	var packagesRaw string
	cmd := captain.NewCommand(
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
				Name:        "force",
				Description: locale.Tl("package_flag_force_description", "Ignore security policy preventing packages with CVEs from being installed (not recommended)"),
				Value:       &force,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_nameversion"),
				Description: locale.T("package_arg_nameversion_wildcard_description"),
				Value:       &packagesRaw,
				Required:    true,
			},
		},
		func(_ *captain.Command, args []string) error {
			for _, p := range args {
				if _, err := params.Packages.Add(p); err != nil {
					return locale.WrapInputError(err, "err_install_packages_args", "Invalid install arguments")
				}
			}
			if force {
				prime.Prompt().EnableForce()
			}
			return runner.Run(params)
		},
	)

	cmd.SetGroup(PackagesGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetHasVariableArguments()

	return cmd
}

func newUninstallCommand(prime *primer.Values) *captain.Command {
	runner := uninstall.New(prime, model.NamespacePackage)

	params := uninstall.Params{}

	var packagesRaw string
	cmd := captain.NewCommand(
		"uninstall",
		locale.Tl("package_uninstall_title", "Uninstalling Package"),
		locale.T("package_uninstall_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("package_arg_name"),
				Description: locale.T("package_arg_name_description"),
				Value:       &packagesRaw,
				Required:    true,
			},
		},
		func(_ *captain.Command, args []string) error {
			for _, p := range args {
				if _, err := params.Packages.Add(p); err != nil {
					return locale.WrapInputError(err, "err_uninstall_packages_args", "Invalid package uninstall arguments")
				}
			}
			return runner.Run(params)
		},
	)

	cmd.SetGroup(PackagesGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetHasVariableArguments()

	return cmd
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
