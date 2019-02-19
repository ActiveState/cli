package organizations

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientOrgs "github.com/ActiveState/cli/pkg/platform/api/client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchAll fetches all organizations for the current user.
func FetchAll() ([]*models.Organization, *failures.Failure) {
	params := clientOrgs.NewListOrganizationsParams()
	memberOnly := true
	personal := false
	params.SetMemberOnly(&memberOnly)
	params.SetPersonal(&personal)
	res, err := authentication.Client().Organizations.ListOrganizations(params, authentication.ClientAuth())

	if err != nil {
		return nil, processErrorResponse(err)
	}

	return res.Payload, nil
}

// FetchByURLName fetches an organization accessible to the current user by it's URL Name.
func FetchByURLName(urlName string) (*models.Organization, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationName = urlName
	resOk, err := authentication.Client().Organizations.GetOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, processErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchMembers fetches the members of an organization accessible to the current user by it's URL Name.
func FetchMembers(urlName string) ([]*models.Member, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationMembersParams()
	params.OrganizationName = urlName
	resOk, err := authentication.Client().Organizations.GetOrganizationMembers(params, authentication.ClientAuth())
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
		return api.FailOrganizationNotFound.New("err_api_org_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
}
