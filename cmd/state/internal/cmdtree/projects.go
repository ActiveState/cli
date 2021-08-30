package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/projects"
)

func newProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewProjects(prime)
	params := projects.NewParams()

	return captain.NewCommand(
		"projects",
		locale.Tl("projects_title", "Listing Projects"),
		locale.T("projects_description"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params)
		},
	).SetGroup(EnvironmentGroup)
}

func newRemoteProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewProjects(prime)
	params := projects.NewParams()

	return captain.NewCommand(
		"remote",
		locale.Tl("projects_remote_title", "Listing Remote Projects"),
		locale.Tl("projects_remote_description", "Manage all projects, including ones you have not checked out locally"),
		prime.Output(),
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.RunRemote(params)
		},
	).SetGroup(EnvironmentGroup)
}
