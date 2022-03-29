package fork

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Namespace    project.Namespaced
	Organization string
	Name         string
	Private      bool
}

type Fork struct {
	out    output.Outputer
	auth   *authentication.Auth
	prompt prompt.Prompter
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
}

func New(prime primeable) *Fork {
	return &Fork{
		prime.Output(),
		prime.Auth(),
		prime.Prompt(),
	}
}

type outputFormat struct {
	Message string
	source  *project.Namespaced
	target  *project.Namespaced
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.EditorV0FormatName:
		return f.editorV0Format()
	}

	return f.Message
}

func (f *Fork) Run(params *Params) error {
	err := f.run(params)

	// Rather than having special error handling for each error we return, just wrap them here
	if err != nil && f.out.Type() == output.EditorV0FormatName {
		return &editorV0Error{err}
	}

	return err
}

func (f *Fork) run(params *Params) error {
	if !f.auth.Authenticated() {
		return locale.NewInputError("err_auth_required", "Authentication is required, please authenticate by running 'state auth'")
	}

	target := &project.Namespaced{
		Owner:   params.Organization,
		Project: params.Name,
	}

	if target.Owner == "" {
		var err error
		target.Owner, err = determineOwner(f.auth.WhoAmI(), f.prompt)
		if err != nil {
			return errs.Wrap(err, "Cannot continue without an owner")
		}
	}

	if target.Project == "" {
		target.Project = params.Namespace.Project
	}

	f.out.Notice(locale.Tl("fork_forking", "Creating fork of {{.V0}} at https://{{.V1}}/{{.V2}}..", params.Namespace.String(), constants.PlatformURL, target.String()))

	_, err := model.CreateCopy(params.Namespace.Owner, params.Namespace.Project, target.Owner, target.Project, params.Private)
	if err != nil {
		return locale.WrapError(err, "err_fork_project", "Could not successfully create fork")
	}

	f.out.Print(&outputFormat{
		locale.Tl("fork_success", "Your fork has been successfully created at https://{{.V0}}/{{.V1}}.", constants.PlatformURL, target.String()),
		&params.Namespace,
		target,
	})

	return nil
}

func determineOwner(username string, prompter prompt.Prompter) (string, error) {
	orgs, err := model.FetchOrganizations()
	if err != nil {
		return "", locale.WrapError(err, "err_fork_orgs", "Could not retrieve list of organizations that you belong to.")
	}
	if len(orgs) == 0 {
		return username, nil
	}

	options := make([]string, len(orgs))
	for i, org := range orgs {
		options[i] = org.DisplayName
	}
	options = append([]string{username}, options...)

	r, err := prompter.Select(locale.Tl("fork_owner_title", "Owner"), locale.Tl("fork_select_org", "Who should the new project belong to?"), options, new(string))
	return r, err
}
