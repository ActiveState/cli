package fork

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api"
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

func (f *Fork) Run(params *Params) error {
	if !f.auth.Authenticated() {
		return locale.NewInputError("err_auth_required", "Authentication is required. Please authenticate by running 'state auth'")
	}

	target := &project.Namespaced{
		Owner:   params.Organization,
		Project: params.Name,
	}

	if target.Owner == "" {
		var err error
		target.Owner, err = determineOwner(f.auth.WhoAmI(), f.prompt, f.auth)
		if err != nil {
			return errs.Wrap(err, "Cannot continue without an owner")
		}
	}

	if target.Project == "" {
		target.Project = params.Namespace.Project
	}

	url := api.GetPlatformURL(target.String()).String()

	f.out.Notice(locale.Tl("fork_forking", "Creating fork of {{.V0}} at {{.V1}}...", params.Namespace.String(), url))

	_, err := model.CreateCopy(params.Namespace.Owner, params.Namespace.Project, target.Owner, target.Project, params.Private, f.auth)
	if err != nil {
		return locale.WrapError(err, "err_fork_project", "Could not create fork")
	}

	f.out.Print(output.Prepare(
		locale.Tl("fork_success", "Your fork has been successfully created at {{.V0}}.", url),
		&struct {
			OriginalOwner string `json:"OriginalOwner"`
			OriginalName  string `json:"OriginalName"`
			NewOwner      string `json:"NewOwner"`
			NewName       string `json:"NewName"`
		}{
			params.Namespace.Owner,
			params.Namespace.Project,
			target.Owner,
			target.Project,
		}))

	return nil
}

func determineOwner(username string, prompter prompt.Prompter, auth *authentication.Auth) (string, error) {
	orgs, err := model.FetchOrganizations(auth)
	if err != nil {
		return "", locale.WrapError(err, "err_fork_orgs", "Could not retrieve list of organizations that you belong to.")
	}
	if len(orgs) == 0 {
		return username, nil
	}

	options := make([]string, len(orgs))
	displayNameToURLNameMap := make(map[string]string)
	for i, org := range orgs {
		options[i] = org.DisplayName
		displayNameToURLNameMap[org.DisplayName] = org.URLname
	}
	options = append([]string{username}, options...)

	r, err := prompter.Select(locale.Tl("fork_owner_title", "Owner"), locale.Tl("fork_select_org", "Who should the new project belong to?"), options, ptr.To(""), nil)
	owner, exists := displayNameToURLNameMap[r]
	if !exists {
		return "", errs.New("Selected organization does not have a URL name")
	}
	return owner, err
}
