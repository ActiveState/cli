package projects

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
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

func (e *EditParams) validate() error {
	if e.Visibility != "" &&
		!strings.EqualFold(e.Visibility, visibilityPublic) &&
		!strings.EqualFold(e.Visibility, visibilityPrivate) {
		return locale.NewInputError("err_edit_visibility", "Visibility must be either public or private")
	}

	return nil
}

type Edit struct {
	auth   *authentication.Auth
	out    output.Outputer
	prompt prompt.Prompter
	config configGetter
	svcm   *model.SvcModel
}

func NewEdit(prime primeable) *Edit {
	return &Edit{
		auth:   prime.Auth(),
		out:    prime.Output(),
		prompt: prime.Prompt(),
		config: prime.Config(),
		svcm:   prime.SvcModel(),
	}
}

func (e *Edit) Run(params *EditParams) error {
	if !e.auth.Authenticated() {
		return locale.NewInputError("err_project_edit_not_authenticated", "In order to edit your project you need to be authenticated. Please run '[ACTIONABLE]state auth[/RESET]' to authenticate.")
	}

	err := params.validate()
	if err != nil {
		return locale.WrapError(err, "err_edit_invalid_params", "Invalid edit parameters")
	}

	editMsg := locale.Tl("edit_prompt", "You are about to edit the following fields for the project [NOTICE]{{.V0}}[/RESET]:\n", params.Namespace.String())
	editable := &mono_models.ProjectEditable{}
	if params.ProjectName != "" {
		editMsg += locale.Tl("edit_prompt_name", "  - Name: {{.V0}}\n", params.ProjectName)
		editable.Name = params.ProjectName
	}

	if params.Visibility != "" {
		editMsg += locale.Tl("edit_prompt_visibility", "  - Visibility: {{.V0}}\n", params.Visibility)
		if strings.EqualFold(params.Visibility, visibilityPublic) {
			editable.Private = ptr.To(false)
		} else {
			editable.Private = ptr.To(true)
		}
	}

	if params.Repository != "" {
		editMsg += locale.Tl("edit_prompt_repo", "  - Repository: {{.V0}}\n", params.Repository)
		editable.RepoURL = &params.Repository
	}

	if params.ProjectName != "" {
		editMsg += locale.Tr("edit_prompt_name_notice", params.Namespace.Owner, params.ProjectName)
	}

	editMsg += locale.Tl("edit_prompt_confirm", "Continue?")

	defaultChoice := !e.out.Config().Interactive
	edit, err := e.prompt.Confirm("", editMsg, &defaultChoice)
	if err != nil {
		return locale.WrapError(err, "err_edit_prompt", "Could not prompt for edit confirmation")
	}

	if !edit {
		e.out.Print(locale.Tl("edit_cancelled", "Project edit cancelled"))
		return nil
	}

	if err = model.EditProject(params.Namespace.Owner, params.Namespace.Project, editable, e.auth); err != nil {
		return locale.WrapError(err, "err_edit_project", "Could not edit project")
	}

	if err = e.editLocalCheckouts(params); err != nil {
		return locale.WrapError(err, "err_edit_local_checkouts")
	}

	if err = e.updateProjectMapping(params); err != nil {
		return locale.WrapError(err, "err_edit_project_mapping")
	}

	e.out.Notice(locale.Tl("edit_success", "Project edited successfully"))

	return nil
}

func (e *Edit) editLocalCheckouts(params *EditParams) error {
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
			return errs.Wrap(err, "Could not edit local checkout at %s", checkout)
		}
	}

	return nil
}

func (e *Edit) editLocalCheckout(owner, checkout string, params *EditParams) error {
	if params.ProjectName == "" {
		return nil
	}

	pjFile, err := projectfile.FromPath(checkout)
	if err != nil {
		return errs.Wrap(err, "Could not get projectfile at %s", checkout)
	}

	info := checkoutinfo.New(e.auth, e.config, pjFile, e.svcm)
	err = info.SetNamespace(owner, params.ProjectName)
	if err != nil {
		return errs.Wrap(err, "Could not set project namespace at %s", checkout)
	}

	return nil
}

func (e *Edit) updateProjectMapping(params *EditParams) error {
	if params.ProjectName == "" {
		return nil
	}

	localProjects := projectfile.GetStaleProjectMapping(e.config)

	var localCheckouts []string
	for namespace, checkouts := range localProjects {
		if namespace == params.Namespace.String() {
			localCheckouts = append(localCheckouts, checkouts...)
		}
	}

	ns := project.Namespaced{Owner: params.Namespace.Owner, Project: params.ProjectName}
	for _, checkout := range localCheckouts {
		projectfile.StoreProjectMapping(e.config, ns.String(), checkout)
	}

	return nil
}
