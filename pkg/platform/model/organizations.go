package model

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientOrgs "github.com/ActiveState/cli/pkg/platform/api/client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchOrganizations fetches all organizations for the current user.
func FetchOrganizations() ([]*models.Organization, *failures.Failure) {
	params := clientOrgs.NewListOrganizationsParams()
	memberOnly := true
	personal := false
	params.SetMemberOnly(&memberOnly)
	params.SetPersonal(&personal)
	res, err := authentication.Client().Organizations.ListOrganizations(params, authentication.ClientAuth())

	if err != nil {
		return nil, processOrgErrorResponse(err)
	}

	return res.Payload, nil
}

// FetchOrgByURLName fetches an organization accessible to the current user by it's URL Name.
func FetchOrgByURLName(urlName string) (*models.Organization, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationName = urlName
	resOk, err := authentication.Client().Organizations.GetOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMembers fetches the members of an organization accessible to the current user by it's URL Name.
func FetchOrgMembers(urlName string) ([]*models.Member, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationMembersParams()
	params.OrganizationName = urlName
	resOk, err := authentication.Client().Organizations.GetOrganizationMembers(params, authentication.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMember fetches the member of an organization accessible to the current user by it's URL Name.
func FetchOrgMember(org *models.Organization, name string) (*models.Member, *failures.Failure) {
	members, failure := FetchOrgMembers(org.Urlname)
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

func processOrgErrorResponse(err error) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailOrganizationNotFound.New("err_api_org_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
}
