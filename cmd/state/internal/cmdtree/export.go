package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/export"
	"github.com/ActiveState/cli/internal/runners/export/config"
	"github.com/ActiveState/cli/internal/runners/export/docs"
	"github.com/ActiveState/cli/internal/runners/export/ghactions"
)

func newExportCommand(prime *primer.Values) *captain.Command {
	runner := export.NewExport()

	return captain.NewCommand(
		"export",
		locale.Tl("export_title", "Exporting Information"),
		locale.T("export_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(ccmd)
		}).SetGroup(UtilsGroup)
}

func newRecipeCommand(prime *primer.Values) *captain.Command {
	recipe := export.NewRecipe(prime)

	params := export.RecipeParams{}

	return captain.NewCommand(
		"recipe",
		locale.Tl("export_recipe_title", "Exporting Recipe Data"),
		locale.T("export_recipe_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name:        "pretty",
				Description: locale.T("export_recipe_flag_pretty"),
				Value:       &params.Pretty,
			},
			{
				Name:        "platform",
				Shorthand:   "p",
				Description: locale.T("export_recipe_flag_platform"),
				Value:       &params.Platform,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("export_recipe_cmd_commitid_arg"),
				Description: locale.T("export_recipe_cmd_commitid_arg_description"),
				Value:       &params.CommitID,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return recipe.Run(&params)
		})
}

func newJWTCommand(prime *primer.Values) *captain.Command {
	jwt := export.NewJWT(prime)

	params := export.JWTParams{}

	return captain.NewCommand(
		"jwt",
		locale.Tl("export_jwt_title", "Exporting Credentials"),
		locale.T("export_jwt_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return jwt.Run(&params)
		})
}

func newPrivateKeyCommand(prime *primer.Values) *captain.Command {
	privateKey := export.NewPrivateKey(prime)

	params := export.PrivateKeyParams{}

	return captain.NewCommand(
		"private-key",
		locale.Tl("export_privkey_title", "Exporting Private Key"),
		locale.T("export_privkey_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return privateKey.Run(&params)
		})
}

func newAPIKeyCommand(prime *primer.Values) *captain.Command {
	apikey := export.NewAPIKey(prime)
	params := export.APIKeyRunParams{}

	return captain.NewCommand(
		"new-api-key",
		locale.Tl("export_new_api_key_title", "Exporting New API Key"),
		locale.T("export_new_api_key_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("export_new_api_key_arg_name"),
				Description: locale.T("export_new_api_key_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			params.IsAuthed = prime.Auth().Authenticated
			return apikey.Run(params)
		})
}

func newExportConfigCommand(prime *primer.Values) *captain.Command {
	runner := config.New(prime)
	params := config.ConfigParams{}

	return captain.NewCommand(
		"config",
		locale.Tl("export_config_title", "Exporting Configuration Data"),
		locale.T("export_config_cmd_description"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{
			{
				Name: "filter",
				Description: locale.Tr(
					"export_config_flag_filter_description",
					strings.Join(config.SupportedFilters(), ", "),
				),
				Value: &params.Filter,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(ccmd, &params)
		})
}

func newExportGithubActionCommand(prime *primer.Values) *captain.Command {
	runner := ghactions.New(prime)
	params := ghactions.Params{}

	return captain.NewCommand(
		"github-actions",
		locale.Tl("export_ghactions_title", "Exporting Github Action Workflow"),
		locale.Tl("export_ghactions_description", "Create a github action workflow for your project"),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		})
}

func newExportDocsCommand(prime *primer.Values) *captain.Command {
	runner := docs.New(prime)
	params := docs.Params{}

	cmd := captain.NewCommand(
		"_docs",
		locale.Tl("export_docs_title", "Export state tool command reference in markdown format"),
		locale.Tl("export_docs_description", ""),
		prime.Output(),
		prime.Config(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params, ccmd)
		})

	cmd.SetHidden(true)

	return cmd
}
