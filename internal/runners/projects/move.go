package projects

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type MoveParams struct {
	Namespace *project.Namespaced
	NewOwner  string
}

type Move struct {
	auth   *authentication.Auth
	out    output.Outputer
	prompt prompt.Prompter
	config configGetter
}

func NewMove(prime primeable) *Move {
	return &Move{
		auth:   prime.Auth(),
		out:    prime.Output(),
		prompt: prime.Prompt(),
		config: prime.Config(),
	}
}

func NewMoveParams() *MoveParams {
	return &MoveParams{Namespace: &project.Namespaced{}}
}

func (m *Move) Run(params *MoveParams) error {
	if !m.auth.Authenticated() {
		return locale.NewInputError("err_project_move_auth", "In order to move your project you need to be authenticated. Please run '[ACTIONABLE]state auth[/RESET]' to authenticate.")
	}

	defaultChoice := !m.prompt.IsInteractive()
	move, kind, err := m.prompt.Confirm("", locale.Tr("move_prompt", params.Namespace.String(), params.NewOwner, params.Namespace.Project), &defaultChoice, nil)
	if err != nil {
		return errs.Wrap(err, "Unable to confirm")
	}
	if !move {
		return locale.NewInputError("move_cancelled", "Project move aborted by user")
	}
	if kind == prompt.NonInteractive {
		m.out.Notice(locale.T("prompt_continue_non_interactive"))
	}

	if err = model.MoveProject(params.Namespace.Owner, params.Namespace.Project, params.NewOwner, m.auth); err != nil {
		return locale.WrapError(err, "err_move_project", "Could not move project")
	}

	if err = m.updateLocalCheckouts(params); err != nil {
		return locale.WrapError(err, "err_edit_local_checkouts")
	}

	if err = m.updateProjectMapping(params); err != nil {
		return locale.WrapError(err, "err_edit_project_mapping")
	}

	m.out.Notice(locale.Tl("move_success",
		"Project [NOTICE]{{.V0}}[/RESET] successfully moved to the [NOTICE]{{.V1}}[/RESET] organization",
		params.Namespace.Project, params.NewOwner))

	return nil
}

func (m *Move) updateLocalCheckouts(params *MoveParams) error {
	localProjects := projectfile.GetProjectMapping(m.config)

	var localCheckouts []string
	for namespace, checkouts := range localProjects {
		if namespace == params.Namespace.String() {
			localCheckouts = append(localCheckouts, checkouts...)
		}
	}

	for _, checkout := range localCheckouts {
		err := m.updateLocalCheckout(checkout, params)
		if err != nil {
			return errs.Wrap(err, "Could not update local checkout at %s", checkout)
		}
	}

	return nil
}

func (m *Move) updateLocalCheckout(checkout string, params *MoveParams) error {
	pjFile, err := projectfile.FromPath(checkout)
	if err != nil {
		return errs.Wrap(err, "Could not get projectfile at %s", checkout)
	}

	err = pjFile.SetNamespace(params.NewOwner, params.Namespace.Project)
	if err != nil {
		return errs.Wrap(err, "Could not set project namespace at %s", checkout)
	}

	return nil
}

func (m *Move) updateProjectMapping(params *MoveParams) error {
	localProjects := projectfile.GetStaleProjectMapping(m.config)

	var localCheckouts []string
	for namespace, checkouts := range localProjects {
		if namespace == params.Namespace.String() {
			localCheckouts = append(localCheckouts, checkouts...)
		}
	}

	ns := project.Namespaced{Owner: params.NewOwner, Project: params.Namespace.Project}
	for _, checkout := range localCheckouts {
		projectfile.StoreProjectMapping(m.config, ns.String(), checkout)
	}

	return nil
}
