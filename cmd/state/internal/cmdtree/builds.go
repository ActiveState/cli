package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/builds"
	"github.com/ActiveState/cli/pkg/project"
)

func newBuildsCommand(prime *primer.Values) *captain.Command {
	runner := builds.New(prime)
	params := &builds.Params{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"builds",
		locale.Tl("builds_title", "Builds"),
		locale.Tl("builds_description", "Inspect builds created for your project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "all",
				Description: locale.Tl("builds_flags_all_description", "List all builds, including individual package artifacts"),
				Value:       &params.All,
			},
			{
				Name:        "namespace",
				Description: locale.Tl("builds_flags_namespace_description", "The namespace of the project to inspect builds for"),
				Value:       params.Namespace,
			},
			{
				Name:        "commit",
				Description: locale.Tl("builds_flags_commit_description", "The commit ID to inspect builds for"),
				Value:       &params.CommitID,
			},
			{
				Name:        "full-id",
				Description: "List builds with their full identifier",
				Value:       &params.Full,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.DeprioritizeInHelpListing()
	return cmd
}

func newBuildsDownloadCommand(prime *primer.Values) *captain.Command {
	runner := builds.NewDownload(prime)
	params := &builds.DownloadParams{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"dl",
		locale.Tl("builds_download_title", "Download build artifacts"),
		locale.Tl("builds_download_description", "Download build artifacts for a given build"),
		prime,
		[]*captain.Flag{
			{
				Name:        "namespace",
				Description: locale.Tl("builds_download_flags_namespace_description", "The namespace of the project to download artifacts from"),
				Value:       params.Namespace,
			},
			{
				Name:        "commit",
				Description: locale.Tl("builds_download_flags_commit_description", "The commit ID to download artifacts from"),
				Value:       &params.CommitID,
			},
		},
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
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetSupportsStructuredOutput()
	return cmd
}
