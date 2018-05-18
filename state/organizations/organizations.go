package organizations

import (
	"github.com/ActiveState/cli/internal/api"
	clientOrgs "github.com/ActiveState/cli/internal/api/client/organizations"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command is the organization command's definition.
var Command = &commands.Command{
	Name:        "organizations",
	Description: "organizations_description",
	Run:         Execute,
}

func fetchOrganizations() (*clientOrgs.ListOrganizationsOK, error) {
	params := clientOrgs.NewListOrganizationsParams()
	memberOnly := true
	personal := false
	params.SetMemberOnly(&memberOnly)
	params.SetPersonal(&personal)
	return api.Client.Organizations.ListOrganizations(params, api.Auth)
}

// Execute the organizations command.
func Execute(cmd *cobra.Command, args []string) {
	orgs, err := fetchOrganizations()
	if err != nil {
		logging.Errorf("Unable to list member organizations: %s", err)
		failures.Handle(err, locale.T("organizations_err"))
		return
	}

	rows := [][]interface{}{}
	for _, org := range orgs.Payload {
		rows = append(rows, []interface{}{org.Name})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("organization_name")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
