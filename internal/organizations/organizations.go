package organizations

import (
	"strings"

	"github.com/ActiveState/cli/internal/api"
	clientOrgs "github.com/ActiveState/cli/internal/api/client/organizations"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
)

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

func FetchMember(org *models.Organization, name string) (*models.Member, *failures.Failure) {
	members, failure := FetchMembers(org.Urlname)
	if failure != nil {
		return nil, failure
	}

	for _, member := range members {
		if strings.EqualFold(name, member.User.Username) {
			return member, nil
		}
	}
	return nil, api.FailNotFound.New("err_api_member_not_found")
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
