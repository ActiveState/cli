package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/pull"
)

func newPullCommand(prime *primer.Values, globals *globalOptions) *captain.Command {
	runner := pull.New(prime)

	params := &pull.PullParams{}

	return captain.NewCommand(
		"pull",
		locale.Tl("pull_title", "Pulling Remote Project"),
		locale.Tl("pull_description", "Pull in the latest version of your project from the ActiveState Platform"),
		prime,
		[]*captain.Flag{
			{
				Name:        "set-project",
				Shorthand:   "",
				Description: locale.Tl("flag_state_pull_set_project_description", "Pull from the specified project instead of the checked out one"),
				Value:       &params.SetProject,
			},
		},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			params.Force = globals.NonInteractive
			return runner.Run(params)
		}).SetGroup(VCSGroup)
}
