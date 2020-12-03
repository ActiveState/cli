package model

import (
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	clientProjects "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailNoValidProject is a failure for the call api.GetProject
	FailNoValidProject = failures.Type("model.fail.novalidproject", failures.FailUser)

	// FailNoDefaultBranch is a failure in getting a project's default branch
	FailNoDefaultBranch = failures.Type("model.fail.nodefaultbranch")

	// FailCannotConvertModel is a failure to convert a new model to an existing model
	FailCannotConvertModel = failures.Type("model.fail.cannotconvertmodel")

	// FailProjectNameConflict is a failure due to a project name conflict
	FailProjectNameConflict = failures.Type("model.fail.projectconflict")

	// FailProjectNotFound is a fialure due to a project not being found
	FailProjectNotFound = failures.Type("model.fail.projectnotfound", failures.FailNonFatal)
)

// FetchProjectByName fetches a project for an organization.
func FetchProjectByName(orgName string, projectName string) (*mono_models.Project, error) {
	logging.Debug("fetching project (%s) in organization (%s)", projectName, orgName)

	request := request.ProjectByOrgAndName(orgName, projectName)

	gql := graphql.Get()
	response := model.Projects{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, errs.Wrap(err, "GraphQL request failed")
	}

	if len(response.Projects) == 0 {
		if !authentication.Get().Authenticated() {
			return nil, locale.NewInputError("err_api_project_not_found_unauthenticated", "", orgName, projectName)
		}
		return nil, locale.NewInputError("err_api_project_not_found", "", projectName, orgName)
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

// DefaultLanguageForProject fetches the default language belonging to the given project
func DefaultLanguageForProject(orgName, projectName string) (Language, error) {
	languages, fail := FetchLanguagesForProject(orgName, projectName)
	if fail != nil {
		return Language{}, fail
	}

	if len(languages) == 0 {
		return Language{}, locale.NewInputError("err_no_languages")
	}

	return languages[0], nil
}

// LanguageForCommit fetches the name of the language belonging to the given commit
func LanguageForCommit(commitID strfmt.UUID) (string, error) {
	languages, fail := FetchLanguagesForCommit(commitID)
	if fail != nil {
		return "", fail
	}

	if len(languages) == 0 {
		return "", locale.NewInputError("err_no_languages")
	}

	return languages[0].Name, nil
}

// DefaultBranchForProjectName retrieves the default branch for the given project owner/name.
func DefaultBranchForProjectName(owner, name string) (*mono_models.Branch, error) {
	proj, fail := FetchProjectByName(owner, name)
	if fail != nil {
		return nil, fail
	}

	return DefaultBranchForProject(proj)
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

// CreateEmptyProject will create the project on the platform
func CreateEmptyProject(owner, name string, private bool) (*mono_models.Project, error) {
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(owner)
	addParams.SetProject(&mono_models.Project{Name: name, Private: private})
	pj, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		if _, ok := err.(*projects.AddProjectConflict); ok {
			return nil, locale.NewInputError(msg)
		}
		return nil, locale.NewError(msg)
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
		return locale.NewInputError("err_api_not_authenticated")
	case 404:
		p := append([]string{""}, params...)
		return locale.NewInputError("err_api_project_not_found", p...)
	default:
		return locale.WrapError(err, "err_api_unknown", "Unexpected API error")
	}
}
