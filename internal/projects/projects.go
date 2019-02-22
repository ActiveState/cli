package projects

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientProjects "github.com/ActiveState/cli/pkg/platform/api/client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var FailNoDefaultBranch = failures.Type("projects.fail.nodefaultbranch")

// FetchByName fetches a project for an organization.
func FetchByName(orgName string, projectName string) (*models.Project, *failures.Failure) {
	params := clientProjects.NewGetProjectParams()
	params.OrganizationName = orgName
	params.ProjectName = projectName
	resOk, err := authentication.Client().Projects.GetProject(params, authentication.ClientAuth())
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(orgName string) ([]*models.Project, *failures.Failure) {
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(orgName)
	orgProjects, err := authentication.Client().Projects.ListProjects(projParams, authentication.ClientAuth())
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return orgProjects.Payload, nil
}

// DefaultBranch retrieves the default branch for the given project
func DefaultBranch(pj *models.Project) (*models.Branch, *failures.Failure) {
	for _, branch := range pj.Branches {
		if branch.Default {
			return branch, nil
		}
	}
	return nil, FailNoDefaultBranch.New(locale.T("err_no_default_branch"))
}

func processErrorResponse(err error) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailProjectNotFound.New("err_api_project_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
}
