package projects

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/client/organizations"
	clientProjects "github.com/ActiveState/cli/internal/api/client/projects"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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

func fetchProjects() ([]projectWithOrg, error) {
	orgParams := organizations.NewListOrganizationsParams()
	memberOnly := true
	orgParams.SetMemberOnly(&memberOnly)
	orgs, err := api.Client.Organizations.ListOrganizations(orgParams, api.Auth)
	if err != nil {
		logging.Errorf("Unable to fetch member organizations: %s", err)
		return nil, err
	}
	projectsList := []projectWithOrg{}
	for _, org := range orgs.Payload {
		projParams := clientProjects.NewListProjectsParams()
		projParams.SetOrganizationName(org.Name)
		orgProjects, err := api.Client.Projects.ListProjects(projParams, api.Auth)
		if err != nil {
			logging.Errorf("Unable to fetch projects for org %s: %s", org.Name, err)
			return nil, err
		}
		for _, project := range orgProjects.Payload {
			projectsList = append(projectsList, projectWithOrg{project.Name, project.Description, org.Name})
		}
	}
	return projectsList, nil
}

// Execute the projects command.
func Execute(cmd *cobra.Command, args []string) {
	projectsList, err := fetchProjects()
	if err != nil {
		failures.Handle(err, locale.T("project_err"))
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
