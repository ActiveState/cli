package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/pull"
	"github.com/ActiveState/cli/pkg/project"
)

func newPullCommand(pj *project.Project, output output.Outputer) *captain.Command {
	runner := pull.New(pj, output)

	return captain.NewCommand(
		"pull",
		locale.Tl("pull_description", "Pull in the latest version of your project from the ActiveState Platform"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return runner.Run()
		})
}
