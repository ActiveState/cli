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
	).SetGroup(ProjectUsageGroup).SetSupportsStructuredOutput()
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
	).SetGroup(ProjectUsageGroup).SetSupportsStructuredOutput()
}

func newProjectsEditCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewEdit(prime)
	params := &projects.EditParams{
		Namespace: &project.Namespaced{},
	}

	cmd := captain.NewCommand(
		"edit",
		locale.Tl("projects_edit_title", "Edit Project"),
		locale.Tl("projects_edit_description", "Edit the project details for the specified project"),
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
	)

	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetUnstable(true)

	return cmd
}

func newDeleteProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewDelete(prime)
	params := projects.NewDeleteParams()

	cmd := captain.NewCommand(
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
	)
	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetUnstable(true)

	return cmd
}

func newMoveProjectsCommand(prime *primer.Values) *captain.Command {
	runner := projects.NewMove(prime)
	params := projects.NewMoveParams()

	cmd := captain.NewCommand(
		"move",
		locale.Tl("projects_move_title", "Move a project"),
		locale.Tl("projects_move_description", "Move the specified project to another organization"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.Tl("projects_move_namespace", "org/project"),
				Description: locale.Tl("projects_move_namespace_description", "The project to move"),
				Value:       params.Namespace,
				Required:    true,
			},
			{
				Name:        locale.Tl("projects_move_new_org", "new-org-name"),
				Description: locale.Tl("projects_move_org_description", "The organization to move the project to"),
				Value:       &params.NewOwner,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(ProjectUsageGroup)
	cmd.SetUnstable(true)

	return cmd
}
