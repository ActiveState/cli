package projects

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientProjects "github.com/ActiveState/cli/pkg/platform/api/client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

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
