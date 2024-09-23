package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/manifest"
)

func newManifestCommmand(prime *primer.Values) *captain.Command {
	runner := manifest.NewManifest(prime)

	params := manifest.Params{}

	cmd := captain.NewCommand(
		"manifest",
		locale.Tl("manifest_title", "Listing Requirements For Your Project"),
		locale.Tl("manifest_cmd_description", "List the requirements of the current project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "expand",
				Description: locale.Tl("manifest_flag_expand", "Expand requirement names to include their namespace"),
				Value:       &params.Expand,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	cmd.SetGroup(PackagesGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}
