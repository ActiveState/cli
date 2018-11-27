package organizations

import (
	"github.com/ActiveState/cli/internal/api"
	clientOrgs "github.com/ActiveState/cli/internal/api/client/organizations"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command is the organization command's definition.
var Command = &commands.Command{
	Name:        "organizations",
	Aliases:     []string{"orgs"},
	Description: "organizations_description",
	Run:         Execute,
}

// FetchAll fetches all organizations for the current user.
func FetchAll() ([]*models.Organization, *failures.Failure) {
	params := clientOrgs.NewListOrganizationsParams()
	memberOnly := true
	personal := false
	params.SetMemberOnly(&memberOnly)
	params.SetPersonal(&personal)
	res, err := api.Client.Organizations.ListOrganizations(params, api.Auth)

	if err != nil {
		if api.ErrorCode(err) == 401 {
			return nil, api.FailAuth.New("err_api_not_authenticated")
		}
		return nil, api.FailUnknown.Wrap(err)
	}

	return res.Payload, nil
}

// FetchByURLName fetches an organization accssible to the current user by it's URL Name.
func FetchByURLName(urlName string) (*models.Organization, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationName = urlName
	resOk, err := api.Client.Organizations.GetOrganization(params, api.Auth)

	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, api.FailAuth.New("err_api_not_authenticated")
		case 404:
			return nil, api.FailNotFound.New("err_api_org_not_found")
		default:
			return nil, api.FailUnknown.Wrap(err)
		}
	}

	return resOk.Payload, nil
}

// Execute the organizations command.
func Execute(cmd *cobra.Command, args []string) {
	orgs, fail := FetchAll()
	if fail != nil {
		failures.Handle(fail, locale.T("organizations_err"))
		return
	}

	rows := [][]interface{}{}
	for _, org := range orgs {
		rows = append(rows, []interface{}{org.Name})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("organization_name")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
