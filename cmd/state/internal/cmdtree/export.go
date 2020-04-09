package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/export"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func newExportCommand() *captain.Command {
	runner := export.NewExport()

	return captain.NewCommand(
		"export",
		locale.T("export_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(ccmd)
		})
}

func newRecipeCommand() *captain.Command {
	recipe := export.NewRecipe()

	params := export.RecipeParams{}

	return captain.NewCommand(
		"recipe",
		locale.T("export_recipe_cmd_description"),
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

func newJWTCommand() *captain.Command {
	jwt := export.NewJWT()

	params := export.JWTParams{}

	return captain.NewCommand(
		"jwt",
		locale.T("export_jwt_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			params.Auth = authentication.Get()

			return jwt.Run(&params)
		})
}

func newPrivateKeyCommand() *captain.Command {
	privateKey := export.NewPrivateKey()

	params := export.PrivateKeyParams{}

	return captain.NewCommand(
		"private-key",
		locale.T("export_privkey_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			params.Auth = authentication.Get()

			return privateKey.Run(&params)
		})
}

func newAPIKeyCommand(outputer output.Outputer) *captain.Command {
	auth := authentication.Get()

	apikey := export.NewAPIKey(auth, outputer)
	params := export.APIKeyRunParams{}

	return captain.NewCommand(
		"new-api-key",
		locale.T("export_new_api_key_cmd_description"),
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
			params.IsAuthed = auth.Authenticated
			return apikey.Run(params)
		})
}
