package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/packages"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func newBundlesCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewList(prime)

	params := packages.ListRunParams{}

	cmd := captain.NewCommand(
		"bundles",
		locale.Tl("bundles_title", "Listing Bundles"),
		locale.Tl("bundles_cmd_description", "Manage bundles used in your project"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "commit",
				Description: locale.Tl("bundle_list_flag_commit_description", "The commit that the listing should be based on"),
				Value:       &params.Commit,
			},
			{
				Name:        "bundle",
				Description: locale.Tl("bundle_list_flag_name_description", "The filter for the bundles names to include in the listing"),
				Value:       &params.Name,
			},
			{
				Name:        "namespace",
				Description: locale.Tl("namespace_list_flag_bundle_description", "The namespace bundles should be listed from"),
				Value:       &params.Project,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespaceBundle)
		},
	)

	cmd.SetGroup(PackagesGroup)

	return cmd
}

func newBundleInstallCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewInstall(prime)

	params := packages.InstallRunParams{}

	return captain.NewCommand(
		"install",
		locale.Tl("bundle_install_title", "Installing Bundle"),
		locale.Tl("bundle_install_cmd_description", "Add a new bundle to your project"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("bundle_arg_nameversion"),
				Description: locale.T("bundle_arg_nameversion_description"),
				Value:       &params.Package,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespaceBundle)
		},
	)
}

func newBundleUninstallCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewUninstall(prime)

	params := packages.UninstallRunParams{}

	return captain.NewCommand(
		"uninstall",
		locale.Tl("bundle_uninstall_title", "Uninstalling Bundle"),
		locale.Tl("bundle_uninstall_cmd_description", "Remove bundle from your project"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("bundle_arg_name"),
				Description: locale.T("bundle_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespaceBundle)
		},
	)
}

func newBundlesSearchCommand(prime *primer.Values) *captain.Command {
	runner := packages.NewSearch(prime)

	params := packages.SearchRunParams{}

	return captain.NewCommand(
		"search",
		locale.Tl("bundle_search_title", "Searching Bundles"),
		locale.Tl("bundle_search_cmd_description", "Search for all available bundles that can be added to your project"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "language",
				Description: locale.Tl("bundle_search_flag_language_description", "The language used to constrain search results"),
				Value:       &params.Language,
			},
			{
				Name:        "exact-term",
				Description: locale.Tl("bundle_search_flag_exact-term_description", "Ensure that search results match search term exactly"),
				Value:       &params.ExactTerm,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("bundle_arg_name"),
				Description: locale.T("bundle_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params, model.NamespaceBundle)
		},
	)
}
