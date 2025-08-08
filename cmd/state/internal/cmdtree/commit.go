package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/commit"
)

func newCommitCommand(prime *primer.Values) *captain.Command {
	runner := commit.New(prime)

	params := &commit.Params{}

	cmd := captain.NewCommand(
		"commit",
		locale.Tl("commit_title", "Commit Changes"),
		locale.Tl("commit_description", "Commit changes to the Build Script"),
		prime,
		[]*captain.Flag{
			{
				Name:        "ts",
				Description: locale.T("package_flag_ts_description"),
				Value:       &params.Timestamp,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	cmd.SetGroup(EnvironmentSetupGroup)
	cmd.SetSupportsStructuredOutput()

	return cmd
}
