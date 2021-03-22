package model

import (
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	clientProjects "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type ErrProjectNameConflict struct{ *locale.LocalizedError }

type ErrProjectNotFound struct{ *locale.LocalizedError }

// FetchProjectByName fetches a project for an organization.
func FetchProjectByName(orgName string, projectName string) (*mono_models.Project, error) {
	logging.Debug("fetching project (%s) in organization (%s)", projectName, orgName)

	request := request.ProjectByOrgAndName(orgName, projectName)

	gql := graphql.New()
	response := model.Projects{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, errs.Wrap(err, "GraphQL request failed")
	}

	if len(response.Projects) == 0 {
		if !authentication.Get().Authenticated() {
			return nil, locale.NewInputError("err_api_project_not_found_unauthenticated", "", orgName, projectName)
		}
		return nil, &ErrProjectNotFound{locale.NewInputError("err_api_project_not_found", "", projectName, orgName)}
	}

	return response.Projects[0].ToMonoProject()
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(orgName string) ([]*mono_models.Project, error) {
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(orgName)
	orgProjects, err := authentication.Client().Projects.ListProjects(projParams, authentication.ClientAuth())
	if err != nil {
		return nil, processProjectErrorResponse(err)
	}
	return orgProjects.Payload, nil
}

func LanguageByCommit(commitID strfmt.UUID) (Language, error) {
	languages, err := FetchLanguagesForCommit(commitID)
	if err != nil {
		return Language{}, err
	}

	if len(languages) == 0 {
		return Language{}, locale.NewInputError("err_no_languages")
	}

	return languages[0], nil
}

// LanguageForCommit fetches the name of the language belonging to the given commit
func LanguageForCommit(commitID strfmt.UUID) (string, error) {
	languages, err := FetchLanguagesForCommit(commitID)
	if err != nil {
		return "", err
	}

	if len(languages) == 0 {
		return "", locale.NewInputError("err_no_languages")
	}

	return languages[0].Name, nil
}

// DefaultBranchForProjectName retrieves the default branch for the given project owner/name.
func DefaultBranchForProjectName(owner, name string) (*mono_models.Branch, error) {
	proj, err := FetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}

	return DefaultBranchForProject(proj)
}

func BranchesForProject(owner, name string) ([]*mono_models.Branch, error) {
	proj, err := FetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}
	return proj.Branches, nil
}

func BranchNamesForProjectFiltered(owner, name string, excludes ...string) ([]string, error) {
	proj, err := FetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}
	branches := make([]string, 0)
	for _, branch := range proj.Branches {
		for _, exclude := range excludes {
			if branch.Label != exclude {
				branches = append(branches, branch.Label)
			}
		}
	}
	return branches, nil
}

// DefaultBranchForProject retrieves the default branch for the given project
func DefaultBranchForProject(pj *mono_models.Project) (*mono_models.Branch, error) {
	for _, branch := range pj.Branches {
		if branch.Default {
			return branch, nil
		}
	}
	return nil, locale.NewError("err_no_default_branch")
}

// BranchForProjectByName retrieves the named branch for the given project, or
// falls back to the default
func BranchForProjectByName(pj *mono_models.Project, name string) (*mono_models.Branch, error) {
	if name == "" {
		return nil, locale.NewInputError("err_empty_branch", "Empty branch name provided.")
	}

	for _, branch := range pj.Branches {
		if branch.Label != "" && branch.Label == name {
			return branch, nil
		}
	}

	return nil, locale.NewInputError(
		"err_no_matching_branch_label",
		"This project has no branch with label matching [NOTICE]{{.V0}}[/RESET].",
		name,
	)
}

// CreateEmptyProject will create the project on the platform
func CreateEmptyProject(owner, name string, private bool) (*mono_models.Project, error) {
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(owner)
	addParams.SetProject(&mono_models.Project{Name: name, Private: private})
	pj, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		if _, ok := err.(*projects.AddProjectConflict); ok {
			return nil, &ErrProjectNameConflict{locale.WrapInputError(err, msg)}
		}
		return nil, locale.WrapError(err, msg)
	}

	return pj.Payload, nil
}

// MakeProjectPrivate turns the given project private
func MakeProjectPrivate(owner, name string) error {
	editParams := projects.NewEditProjectParams()
	yes := true
	editParams.SetProject(&mono_models.ProjectEditable{
		Private: &yes,
	})
	editParams.SetOrganizationName(owner)
	editParams.SetProjectName(name)

	_, err := authentication.Client().Projects.EditProject(editParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return locale.WrapError(err, msg)
	}

	return nil
}

// ProjectURL creates a valid platform URL for the given project parameters
func ProjectURL(owner, name, commitID string) string {
	url := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, owner, name)
	if commitID != "" {
		url = url + "?commitID=" + commitID
	}
	return url
}

func processProjectErrorResponse(err error, params ...string) error {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return locale.WrapInputError(err, "err_api_not_authenticated")
	case 404:
		p := append([]string{""}, params...)
		return &ErrProjectNotFound{locale.WrapInputError(err, "err_api_project_not_found", p...)}
	default:
		return locale.WrapError(err, "err_api_unknown", "Unexpected API error")
	}
}

func AddBranch(projectID strfmt.UUID, label string) (strfmt.UUID, error) {
	var branchID strfmt.UUID
	addParams := projects.NewAddBranchParams()
	addParams.SetProjectID(projectID)
	addParams.Body.Label = label

	res, err := authentication.Client().Projects.AddBranch(addParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return branchID, locale.WrapError(err, msg)
	}

	return res.Payload.BranchID, nil
}
