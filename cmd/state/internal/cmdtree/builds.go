package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/builds"
)

func newBuildsCommand(prime *primer.Values) *captain.Command {
	runner := builds.New(prime)
	params := &builds.Params{}

	cmd := captain.NewCommand(
		"builds",
		locale.Tl("builds_title", "Builds"),
		locale.Tl("builds_description", "Inspect builds created for your project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "all",
				Description: "List all builds, including individual package artifacts",
				Value:       &params.All,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)
	cmd.DeprioritizeInHelpListing()
	return cmd
}

func newBuildsDownloadCommand(prime *primer.Values) *captain.Command {
	runner := builds.NewDownload(prime)
	params := &builds.DownloadParams{}

	cmd := captain.NewCommand(
		"dl",
		locale.Tl("builds_download_title", "Download build artifacts"),
		locale.Tl("builds_download_description", "Download build artifacts for a given build"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "ID",
				Description: locale.Tl("builds_download_arg_id", "The ID of the artifact to download"),
				Value:       &params.BuildID,
				Required:    true,
			},
			{
				Name:        "target",
				Description: locale.Tl("builds_download_arg_target", "The target directory to download the artifact to"),
				Value:       &params.OutputDir,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)
	cmd.DeprioritizeInHelpListing()
	return cmd
}
