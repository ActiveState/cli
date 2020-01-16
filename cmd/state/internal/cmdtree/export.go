package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
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

type RecipeArgs struct {
	CommitID string
}

type RecipeFlags struct {
	Pretty   bool
	Platform string
}

func newRecipeCommand() *captain.Command {
	recipe := export.NewRecipe()

	args := RecipeArgs{}
	flags := RecipeFlags{}

	return captain.NewCommand(
		"recipe",
		locale.T("export_recipe_cmd_description"),
		[]*captain.Flag{
			{
				Name:        "pretty",
				Description: locale.T("export_recipe_flag_pretty"),
				Value:       &flags.Pretty,
			},
			{
				Name:        "platform",
				Shorthand:   "p",
				Description: locale.T("export_recipe_flag_platform"),
				Value:       &flags.Platform,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("export_recipe_cmd_commitid_arg"),
				Description: locale.T("export_recipe_cmd_commitid_arg_description"),
				Variable:    &args.CommitID,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return recipe.Run(&export.RecipeParams{
				CommitID: args.CommitID,
				Pretty:   flags.Pretty,
				Platform: flags.Platform,
			})
		})
}

func newJWTCommand() *captain.Command {
	jwt := export.NewJWT()

	return captain.NewCommand(
		"jwt",
		locale.T("export_jwt_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return jwt.Run(&export.JWTParams{Auth: authentication.Get()})
		})
}

func newPrivateKeyCommand() *captain.Command {
	privateKey := export.NewPrivateKey()

	return captain.NewCommand(
		"private-key",
		locale.T("export_privkey_cmd_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return privateKey.Run(&export.PrivateKeyParams{Auth: authentication.Get()})
		})
}
