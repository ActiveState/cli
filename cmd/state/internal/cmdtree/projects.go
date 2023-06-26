package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/projects"
	"github.com/ActiveState/cli/pkg/project"
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

func newProjectsEditCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewEdit(prime)
	params := projects.EditParams{
		Namespace: &project.Namespaced{},
	}

	return captain.NewCommand(
		"edit",
		locale.Tl("projects_edit_title", "Edit Project"),
		locale.T("projects_edit_description"),
		prime,
		[]*captain.Flag{
			{
				Name:        "name",
				Description: locale.Tl("projects_edit_name_description", "Edit the name of the project."),
				Value:       &params.ProjectName,
			},
			{
				Name:        "visibility",
				Description: locale.Tl("projects_edit_visibility_description", "Edit the visibility to non-members, either public or private."),
				Value:       &params.Visibility,
			},
			{
				Name:        "repository",
				Description: locale.Tl("projects_edit_repository_description", "Edit the linked VCS repo. To unset use --repo=\"\"."),
				Value:       &params.Repository,
			},
		},
		[]*captain.Argument{
			{
				Name:        "namespace",
				Description: locale.Tl("projects_edit_namespace_description", "The namespace of the project to edit"),
				Required:    true,
				Value:       params.Namespace,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params)
		},
	).SetGroup(ProjectUsageGroup).SetUnstable(true)
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
	).SetGroup(ProjectUsageGroup).SetUnstable(true)
}
