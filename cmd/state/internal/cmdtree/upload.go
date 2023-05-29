package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/uploadingredient"
)

func newUpload(prime *primer.Values) *captain.Command {
	runner := uploadingredient.New(prime)
	params := uploadingredient.Params{}
	c := captain.NewCommand(
		"upload",
		locale.Tl("add_title", "Uploading Ingredient"),
		locale.Tl("add_description", "Upload an Ingredient for private consumption."),
		prime,
		[]*captain.Flag{
			{
				Name:        "edit",
				Description: locale.Tl("author_upload_edit_description", "Create a revision for an existing ingredient, matched by their name and namespace."),
				Value:       &params.Edit,
			},
			{
				Name:        "editor",
				Description: locale.Tl("author_upload_editor_description", "Edit the ingredient information in your editor before uploading."),
				Value:       &params.Editor,
			},
			{
				Name: "name",
				Description: locale.Tl(
					"author_upload_name_description",
					"The name of the ingredient. Defaults to basename of filepath.",
				),
				Value: &params.Name,
			},
			{
				Name: "version",
				Description: locale.Tl(
					"author_upload_version_description",
					"Version of the ingredient (preferably semver).",
				),
				Value: &params.Version,
			},
			{
				Name: "namespace",
				Description: locale.Tl(
					"author_upload_name_description",
					"The namespace of the ingredient. Defaults to <orgname>/shared. Must start with '<orgname>/'.",
				),
				Value: &params.Namespace,
			},
			{
				Name: "description",
				Description: locale.Tl(
					"author_upload_description_description",
					"A short description summarizing what this ingredient is for.",
				),
				Value: &params.Description,
			},
			{
				Name: "author",
				Description: locale.Tl(
					"author_upload_author_description",
					"Ingredient author, in the format of \"[<name>] <email>\". Can be set multiple times.",
				),
				Value: &params.Authors,
			},
			{
				Name: "depend",
				Description: locale.Tl(
					"author_upload_depend_description",
					"Ingredient that this ingredient depends on, format as <namespace>/<name>@<version>. Can be set multiple times.",
				),
				Value: &params.Depends,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.Tl("filepath", "filepath"),
				Description: locale.Tl("author_upload_filepath_description", "The binary ingredient file to upload."),
				Value:       &params.Filepath,
				Required:    true,
			},
			{
				Name:        locale.Tl("meta_file", "meta file"),
				Description: locale.Tl("author_upload_metafile_description", "A yaml file expressing the ingredient meta information. Use --editor to review the file format."),
				Value:       &params.MetaFilepath,
				Required:    false,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(&params)
		})
	c.SetGroup(AuthorGroup)
	c.SetUnstable(true)
	return c
}
