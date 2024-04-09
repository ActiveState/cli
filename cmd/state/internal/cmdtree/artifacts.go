package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/artifacts"
	"github.com/ActiveState/cli/pkg/project"
)

func newArtifactsCommand(prime *primer.Values) *captain.Command {
	runner := artifacts.New(prime)
	params := &artifacts.Params{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"artifacts",
		locale.Tl("artifacts_title", "Artifacts"),
		locale.Tl("artifacts_description", "Inspect artifacts created for your project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "all",
				Description: locale.Tl("artifacts_flags_all_description", "List all artifacts, including individual package artifacts"),
				Value:       &params.All,
			},
			{
				Name:        "namespace",
				Description: locale.Tl("artifacts_flags_namespace_description", "The namespace of the project to inspect artifacts for"),
				Value:       params.Namespace,
			},
			{
				Name:        "commit",
				Description: locale.Tl("artifacts_flags_commit_description", "The commit ID to inspect artifacts for"),
				Value:       &params.CommitID,
			},
			{
				Name:        "target",
				Description: locale.Tl("artifacts_flags_target_description", "The target to report artifacts for"),
				Value:       &params.Target,
			},
			{
				Name:        "full-id",
				Description: "List artifacts with their full identifier",
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
	cmd.SetAliases("builds")
	return cmd
}

func newArtifactsDownloadCommand(prime *primer.Values) *captain.Command {
	runner := artifacts.NewDownload(prime)
	params := &artifacts.DownloadParams{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"dl",
		locale.Tl("artifacts_download_title", "Download build artifacts"),
		locale.Tl("artifacts_download_description", "Download build artifacts for a given build"),
		prime,
		[]*captain.Flag{
			{
				Name:        "namespace",
				Description: locale.Tl("artifacts_download_flags_namespace_description", "The namespace of the project to download artifacts from"),
				Value:       params.Namespace,
			},
			{
				Name:        "commit",
				Description: locale.Tl("artifacts_download_flags_commit_description", "The commit ID to download artifacts from"),
				Value:       &params.CommitID,
			},
			{
				Name:        "target",
				Description: locale.Tl("artifacts_flags_target_description", "The target to download artifacts from"),
				Value:       &params.Target,
			},
		},
		[]*captain.Argument{
			{
				Name:        "ID",
				Description: locale.Tl("artifacts_download_arg_id", "The ID of the artifact to download"),
				Value:       &params.BuildID,
				Required:    true,
			},
			{
				Name:        "path",
				Description: locale.Tl("artifacts_download_arg_target", "The target path to download the artifact to"),
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
