package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/commit"
)

func newCommitCommand(prime *primer.Values) *captain.Command {
	runner := commit.New(prime)

	cmd := captain.NewCommand(
		"commit",
		locale.Tl("commit_title", "Commit Changes"),
		locale.Tl("commit_description", "Commit changes to the Build Script"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run()
		},
	)

	cmd.SetGroup(EnvironmentSetupGroup)
	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}
