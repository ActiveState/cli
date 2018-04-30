package projects

import (
	"github.com/ActiveState/cli/internal/api"
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

func Execute(cmd *cobra.Command, args []string) {
	projects, err := api.Client.Projects.ListProjects(nil, api.Auth)
	if err != nil {
		failures.Handle(err, locale.T("err_api_response"))
		return
	}

	rows := [][]interface{}{}
	for _, project := range projects.Payload {
		rows = append(rows, []interface{}{project.Name, project.Description})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("project_name"), locale.T("project_description")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
