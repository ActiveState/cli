package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/projects"
)

func newProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewProjects(prime)

	return captain.NewCommand(
		"projects",
		locale.T("projects_description"),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run()
		},
	)
}
