package projects

import (
	"github.com/ActiveState/cli/internal/api"
	clientProjects "github.com/ActiveState/cli/internal/api/client/projects"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
)

// FetchByName fetches a project for an organization.
func FetchByName(org *models.Organization, projectName string) (*models.Project, *failures.Failure) {
	params := clientProjects.NewGetProjectParams()
	params.OrganizationName = org.Urlname
	params.ProjectName = projectName
	resOk, err := api.Client.Projects.GetProject(params, api.Auth)
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(org *models.Organization) ([]*models.Project, *failures.Failure) {
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(org.Urlname)
	orgProjects, err := api.Client.Projects.ListProjects(projParams, api.Auth)
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return orgProjects.Payload, nil
}

func processErrorResponse(err error) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailNotFound.New("err_api_project_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
}
