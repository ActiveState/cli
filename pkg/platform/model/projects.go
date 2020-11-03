package model

import (
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
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
func FetchProjectByName(orgName string, projectName string) (*mono_models.Project, *failures.Failure) {
	logging.Debug("fetching project (%s) in organization (%s)", projectName, orgName)

	request := request.ProjectByOrgAndName(orgName, projectName)

	gql := graphql.Get()
	response := model.Projects{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, api.FailUnknown.Wrap(err)
	}

	if len(response.Projects) == 0 {
		if !authentication.Get().Authenticated() {
			return nil, FailNoValidProject.New(locale.Tr("err_api_project_not_found_unauthenticated", orgName, projectName))
		}
		return nil, FailProjectNotFound.New(locale.Tr("err_api_project_not_found", projectName, orgName))
	}

	return response.Projects[0].ToMonoProject()
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(orgName string) ([]*mono_models.Project, *failures.Failure) {
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(orgName)
	orgProjects, err := authentication.Client().Projects.ListProjects(projParams, authentication.ClientAuth())
	if err != nil {
		return nil, processProjectErrorResponse(err)
	}
	return orgProjects.Payload, nil
}

// DefaultLanguageForProject fetches the default language belonging to the given project
func DefaultLanguageForProject(orgName, projectName string) (Language, *failures.Failure) {
	languages, fail := FetchLanguagesForProject(orgName, projectName)
	if fail != nil {
		return Language{}, fail
	}

	if len(languages) == 0 {
		return Language{}, failures.FailUser.New(locale.T("err_no_languages"))
	}

	return languages[0], nil
}

// LanguageForCommit fetches the name of the language belonging to the given commit
func LanguageForCommit(commitID strfmt.UUID) (string, *failures.Failure) {
	languages, fail := FetchLanguagesForCommit(commitID)
	if fail != nil {
		return "", fail
	}

	if len(languages) == 0 {
		return "", failures.FailUser.New(locale.T("err_no_languages"))
	}

	return languages[0].Name, nil
}

// DefaultBranchForProjectName retrieves the default branch for the given project owner/name.
func DefaultBranchForProjectName(owner, name string) (*mono_models.Branch, *failures.Failure) {
	proj, fail := FetchProjectByName(owner, name)
	if fail != nil {
		return nil, fail
	}

	return DefaultBranchForProject(proj)
}

// DefaultBranchForProject retrieves the default branch for the given project
func DefaultBranchForProject(pj *mono_models.Project) (*mono_models.Branch, *failures.Failure) {
	for _, branch := range pj.Branches {
		if branch.Default {
			return branch, nil
		}
	}
	return nil, FailNoDefaultBranch.New(locale.T("err_no_default_branch"))
}

// CreateProject will create the project on the platform
func CreateProject(owner, name, hostPlatform string, lang *language.Supported, langVersion string, private bool) (*mono_models.Project, strfmt.UUID, *failures.Failure) {
	_, fail := CreateEmptyProject(owner, name, private)
	if fail != nil {
		return nil, "", fail
	}

	var requirement string
	if lang != nil {
		requirement = lang.Requirement()
		if langVersion == "" {
			langVersion = lang.RecommendedVersion()
		}
	}

	return CommitInitial(owner, name, hostPlatform, requirement, langVersion)
}

// CreateEmptyProject will create the project on the platform
func CreateEmptyProject(owner, name string, private bool) (*mono_models.Project, *failures.Failure) {
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(owner)
	addParams.SetProject(&mono_models.Project{Name: name, Private: private})
	pj, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		if _, ok := err.(*projects.AddProjectConflict); ok {
			return nil, FailProjectNameConflict.New(msg)
		}
		return nil, api.FailUnknown.New(msg)
	}

	return pj.Payload, nil
}

// MakeProjectPrivate turns the given project private
func MakeProjectPrivate(owner, name string) *failures.Failure {
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
		return api.FailUnknown.Wrap(err, msg)
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

func processProjectErrorResponse(err error, params ...string) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailProjectNotFound.New("err_api_project_not_found", params...)
	default:
		return api.FailUnknown.Wrap(err)
	}
}

func IsProjectModifiable(owner, projectName string) (bool, error) {
	auth := authentication.Get()
	if !auth.Authenticated() {
		return false, nil
	}

	project, fail := FetchProjectByName(owner, projectName)
	if fail != nil {
		return false, fail.ToError()
	}

	org, err := FetchOrganizationByID(project.OrganizationID)
	if err != nil {
		return false, err
	}

	_, fail = FetchOrgMember(org.URLname, auth.WhoAmI())
	if fail != nil {
		if api.FailNotFound.Matches(fail.Type) {
			return false, nil
		}
		return false, fail.ToError()
	}

	return true, nil
}
