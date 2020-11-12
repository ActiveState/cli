package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/revert"
)

func newRevertCommand(prime *primer.Values) *captain.Command {
	runner := revert.New(prime)
	params := &revert.Params{}

	return captain.NewCommand(
		"revert",
		locale.Tl("revert_title", "Reverting Commit"),
		locale.Tl("revert_description", "Revert a commit"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "commit-id",
				Description: locale.Tl("revert_arg_commit_id", "The commit ID to revert to"),
				Required:    true,
				Value:       &params.CommitID,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params)
		},
	).SetGroup(VCSGroup)
}
