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
	Namespace    project.ParsedURL
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
	source  *project.ParsedURL
	target  *project.ParsedURL
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

	target := &project.ParsedURL{
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

	// Retrieve the source project that we'll be forking
	sourceProject, err := model.FetchProjectByName(params.Namespace.Owner, params.Namespace.Project)
	if err != nil {
		return locale.WrapInputError(err, "err_fork_fetchProject", "Could not find the source project: {{.V0}}", params.Namespace.String())
	}

	// Create the target project
	targetProject, err := model.CreateEmptyProject(target.Owner, target.Project, false)
	if err != nil {
		return locale.WrapError(err, "err_fork_createProject", "Could not create project: {{.V0}}", target.String())
	}

	// Set up the forked branch on the target project
	if err := model.TrackBranch(sourceProject, targetProject); err != nil {
		return locale.WrapError(err, "err_fork_track", "Could not set up the forked branch for your new project.")
	}

	// Turn the target project private if this was requested (unfortunately this can't be done int the Creation step)
	if params.Private {
		if err := model.MakeProjectPrivate(target.Owner, target.Project); err != nil {
			return locale.WrapError(
				err, "err_fork_private",
				"Your project was created but could not be made private, please head over to https://{{.V0}}/{{.V1}} to manually update your privacy settings.",
				constants.PlatformURL, target.String())
		}
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
		options[i] = org.Name
	}
	options = append([]string{username}, options...)

	r, err := prompter.Select(locale.Tl("fork_owner_title", "Owner"), locale.Tl("fork_select_org", "Who should the new project belong to?"), options, new(string))
	return r, err
}
