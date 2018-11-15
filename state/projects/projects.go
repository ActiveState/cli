package projects

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/client/organizations"
	clientProjects "github.com/ActiveState/cli/internal/api/client/projects"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "projects",
	Description: "projects_description",
	Run:         Execute,
}

// Holds a union of project and organization parameters.
type projectWithOrg struct {
	Name         string
	Description  string
	Organization string
}

// FetchByName fetches a project for an organization.
func FetchByName(org *models.Organization, projectName string) (*models.Project, *failures.Failure) {
	params := clientProjects.NewGetProjectParams()
	params.OrganizationName = org.Urlname
	params.ProjectName = projectName
	resOk, err := api.Client.Projects.GetProject(params, api.Auth)

	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, api.FailAuth.New("err_api_not_authenticated")
		case 404:
			return nil, api.FailNotFound.New("err_api_project_not_found")
		default:
			return nil, api.FailUnknown.Wrap(err)
		}
	}

	return resOk.Payload, nil
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(org *models.Organization) ([]*models.Project, *failures.Failure) {
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(org.Urlname)
	orgProjects, err := api.Client.Projects.ListProjects(projParams, api.Auth)
	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, api.FailAuth.New("err_api_not_authenticated")
		case 404:
			return nil, api.FailNotFound.New("err_api_project_not_found")
		default:
			return nil, api.FailUnknown.Wrap(err)
		}
	}
	return orgProjects.Payload, nil
}

func fetchProjects() ([]projectWithOrg, *failures.Failure) {
	orgParams := organizations.NewListOrganizationsParams()
	memberOnly := true
	orgParams.SetMemberOnly(&memberOnly)
	orgs, err := api.Client.Organizations.ListOrganizations(orgParams, api.Auth)
	if err != nil {
		if api.ErrorCode(err) == 401 {
			return nil, api.FailAuth.New("err_api_not_authenticated")
		}
		return nil, api.FailUnknown.Wrap(err)
	}
	projectsList := []projectWithOrg{}
	for _, org := range orgs.Payload {
		orgProjects, err := FetchOrganizationProjects(org)
		if err != nil {
			return nil, err
		}
		for _, project := range orgProjects {
			projectsList = append(projectsList, projectWithOrg{project.Name, project.Description, org.Name})
		}
	}
	return projectsList, nil
}

// Execute the projects command.
func Execute(cmd *cobra.Command, args []string) {
	projectsList, fail := fetchProjects()
	if fail != nil {
		failures.Handle(fail, locale.T("project_err"))
		return
	}

	if len(projectsList) == 0 {
		print.Line(locale.T("project_empty"))
		return
	}

	rows := [][]interface{}{}
	for _, project := range projectsList {
		rows = append(rows, []interface{}{project.Name, project.Organization, project.Description})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("project_name"), locale.T("organization_name"), locale.T("project_description")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
