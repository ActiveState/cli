package projects

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

const (
	visibilityPublic  = "Public"
	visibilityPrivate = "Private"
)

type EditParams struct {
	Namespace   *project.Namespaced
	ProjectName string
	Visibility  string
	Repository  string
}

func (e EditParams) validate() error {
	if !strings.EqualFold(e.Visibility, visibilityPublic) && !strings.EqualFold(e.Visibility, visibilityPrivate) {
		return locale.NewInputError("err_edit_visibility", "Visibility must be either public or private")
	}

	return nil
}

type Edit struct {
	auth   *authentication.Auth
	out    output.Outputer
	prompt prompt.Prompter
	config configGetter
}

func NewEdit(prime primeable) *Edit {
	return &Edit{
		auth:   prime.Auth(),
		out:    prime.Output(),
		prompt: prime.Prompt(),
		config: prime.Config(),
	}
}

func (e *Edit) Run(params EditParams) error {
	err := params.validate()
	if err != nil {
		return locale.WrapError(err, "err_edit_invalid_params", "Invalid edit parameters")
	}

	editParams := projects.NewEditProjectParams()
	editParams.SetOrganizationName(params.Namespace.Owner)
	editParams.SetProjectName(params.Namespace.Project)

	editMsg := locale.Tl("edit_prompt", "You are about the edit the following project fields for the project {{.V0}}:\n", params.Namespace.String())
	editable := &mono_models.ProjectEditable{}
	switch {
	case params.ProjectName != "":
		editable.Name = params.ProjectName
		editMsg += locale.Tl("edit_prompt_name", "  - Name: {{.V0}}\n", params.ProjectName)
	case params.Visibility != "":
		if strings.EqualFold(params.Visibility, visibilityPublic) {
			editable.Private = p.BoolP(false)
			editMsg += locale.Tl("edit_prompt_visibility", "  - Visibility: {{.V0}}\n", visibilityPublic)
		} else {
			editable.Private = p.BoolP(true)
			editMsg += locale.Tl("edit_prompt_visibility", "  - Visibility: {{.V0}}\n", visibilityPrivate)
		}
	case params.Repository != "":
		editable.RepoURL = &params.Repository
		editMsg += locale.Tl("edit_prompt_repo", "  - Repository: {{.V0}}\n", params.Repository)
	}

	var edit bool
	edit, err = e.prompt.Confirm("", editMsg, &edit)
	if err != nil {
		return locale.WrapError(err, "err_edit_prompt", "Could not prompt for edit confirmation")
	}

	if !edit {
		e.out.Print(locale.Tl("edit_cancelled", "Project edit cancelled"))
		return nil
	}

	err = model.EditProject(params.Namespace.Owner, params.Namespace.Project, editable)
	if err != nil {
		return locale.WrapError(err, "err_edit_project", "Could not edit project")
	}

	err = e.editLocalCheckouts(params)
	if err != nil {
		return locale.WrapError(err, "err_edit_local_checkouts", "Could not edit local checkouts")
	}

	e.out.Print(locale.Tl("edit_success", "Project edited successfully"))

	return nil
}

func (e *Edit) editLocalCheckouts(params EditParams) error {
	localProjects := projectfile.GetProjectMapping(e.config)

	var localCheckouts []string
	for namespace, checkouts := range localProjects {
		if namespace == params.Namespace.String() {
			localCheckouts = append(localCheckouts, checkouts...)
		}
	}

	for _, checkout := range localCheckouts {
		err := e.editLocalCheckout(params.Namespace.Owner, checkout, params)
		if err != nil {
			return locale.WrapError(err, "err_edit_local_checkout", "Could not edit local checkout at {{.V0}}", checkout)
		}
	}

	return nil
}

func (e *Edit) editLocalCheckout(owner, checkout string, params EditParams) error {
	if params.ProjectName == "" {
		return nil
	}

	pjFile, err := projectfile.FromPath(checkout)
	if err != nil {
		return locale.WrapError(err, "err_edit_local_checkout", "Could not get projectfile at {{.V0}}", checkout)
	}

	err = pjFile.SetNamespace(owner, params.ProjectName)
	if err != nil {
		return locale.WrapError(err, "err_edit_local_checkout", "Could not set project namespace at {{.V0}}", checkout)
	}

	return nil
}
