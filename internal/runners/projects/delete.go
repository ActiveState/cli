package projects

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

type DeleteParams struct {
	Project *project.Namespaced
}

type Delete struct {
	auth   *authentication.Auth
	out    output.Outputer
	prompt prompt.Prompter
}

func NewDeleteParams() *DeleteParams {
	return &DeleteParams{&project.Namespaced{}}
}

func NewDelete(prime primeable) *Delete {
	return &Delete{
		prime.Auth(),
		prime.Output(),
		prime.Prompt(),
	}
}

func (d *Delete) Run(params *DeleteParams) error {
	if !d.auth.Authenticated() {
		return locale.NewInputError("err_projects_delete_authenticated", "You need to be authenticated to delete a project.")
	}

	defaultChoice := !d.out.Config().Interactive
	confirm, err := d.prompt.Confirm("", locale.Tl("project_delete_confim", "Are you sure you want to delete the project {{.V0}}?", params.Project.String()), &defaultChoice)
	if err != nil {
		return locale.WrapError(err, "err_project_delete_confirm", "Could not confirm delete choice")
	}
	if !confirm {
		return locale.NewInputError("err_project_delete_aborted", "Delete aborted by user")
	}

	monoParams := projects.NewDeleteProjectParams()
	monoParams.SetOrganizationName(params.Project.Owner)
	monoParams.SetProjectName(params.Project.Project)

	_, err = d.auth.Client().Projects.DeleteProject(monoParams, d.auth.ClientAuth())
	if err != nil {
		return locale.WrapError(err, "err_projects_delete", "Unable to delete project")
	}

	d.out.Notice(locale.Tl("notice_projects_delete", "Your project was deleted. Please delete any additional copies that are checked out, as they will be inoperable."))

	return nil
}
