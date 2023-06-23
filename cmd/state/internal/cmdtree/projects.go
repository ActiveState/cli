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
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params)
		},
	).SetGroup(ProjectUsageGroup)
}

func newRemoteProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewProjects(prime)
	params := projects.NewParams()

	return captain.NewCommand(
		"remote",
		locale.Tl("projects_remote_title", "Listing Remote Projects"),
		locale.Tl("projects_remote_description", "List all projects, including ones you have not checked out locally"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.RunRemote(params)
		},
	).SetGroup(ProjectUsageGroup)
}

func newDeleteProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewDelete(prime)
	params := projects.NewDeleteParams()

	return captain.NewCommand(
		"delete",
		locale.Tl("projects_delete_title", "Delete a project"),
		locale.Tl("projects_delete_description", "Delete the specified project from the Platform"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "namespace",
				Description: locale.Tl("projects_delete_namespace_description", "org/project"),
				Value:       params.Project,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	).SetGroup(ProjectUsageGroup)
}
