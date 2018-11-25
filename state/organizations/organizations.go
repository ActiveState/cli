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
		return nil, processErrorResponse(err)
	}

	return res.Payload, nil
}

// FetchByURLName fetches an organization accessible to the current user by it's URL Name.
func FetchByURLName(urlName string) (*models.Organization, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationName = urlName
	resOk, err := api.Client.Organizations.GetOrganization(params, api.Auth)
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchMembers fetches the members of an organization accessible to the current user by it's URL Name.
func FetchMembers(urlName string) ([]*models.Member, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationMembersParams()
	params.OrganizationName = urlName
	resOk, err := api.Client.Organizations.GetOrganizationMembers(params, api.Auth)
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return resOk.Payload, nil
}

func processErrorResponse(err error) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailNotFound.New("err_api_org_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
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
