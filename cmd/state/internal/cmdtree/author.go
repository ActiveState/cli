package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/uploadingredient"
)

func newAuthorCommand(prime *primer.Values) *captain.Command {
	c := captain.NewCommand(
		"author",
		"",
		locale.Tl("branch_description", "Author packages and ingredients on the ActiveState Platform"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, _ []string) error {
			return cmd.Usage()
		})
	c.SetGroup(AuthorGroup)
	c.SetUnstable(true)
	c.SetHidden(true)
	return c
}

func newAuthorUpload(prime *primer.Values) *captain.Command {
	runner := uploadingredient.New(prime)
	params := uploadingredient.Params{}
	c := captain.NewCommand(
		"upload",
		locale.Tl("add_title", "Uploading Binary Ingredient"),
		locale.Tl("add_description", "Upload a Binary Ingredient for private consumption"),
		prime,
		[]*captain.Flag{
			{
				Name: "name",
				Description: locale.Tl(
					"author_upload_name_description",
					"The name and optionally version of the ingredient, eg. <name[@version]>. Defaults to basename of filepath.",
				),
				Value: &params.NameVersion,
			},
			{
				Name: "namespace",
				Description: locale.Tl(
					"author_upload_namespace_description",
					"The namespace under which the ingredient should be stored. Defaults to <org>/shared",
				),
				Value: &params.Namespace,
			},
			{
				Name: "platform",
				Description: locale.Tl(
					"author_upload_platform_description",
					"The platform this ingredient is intended for. Defaults to your current platform.",
				),
				Value: &params.Platform,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("filepath", "filepath"),
				Description: locale.Tl("author_upload_filepath_description", "The binary ingredient file to upload."),
				Value:       &params.Filepath,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(&params)
		})
	c.SetGroup(AuthorGroup)
	c.SetUnstable(true)
	c.SetHidden(true)
	return c
}
