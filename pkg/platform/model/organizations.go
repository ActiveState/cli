package model

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientOrgs "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchOrganizations fetches all organizations for the current user.
func FetchOrganizations() ([]*mono_models.Organization, *failures.Failure) {
	params := clientOrgs.NewListOrganizationsParams()
	memberOnly := true
	params.SetMemberOnly(&memberOnly)
	res, err := authentication.Client().Organizations.ListOrganizations(params, authentication.ClientAuth())

	if err != nil {
		return nil, processOrgErrorResponse(err)
	}

	return res.Payload, nil
}

// FetchOrgByURLName fetches an organization accessible to the current user by it's URL Name.
func FetchOrgByURLName(urlName string) (*mono_models.Organization, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationName = urlName
	resOk, err := authentication.Client().Organizations.GetOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMembers fetches the members of an organization accessible to the current user by it's URL Name.
func FetchOrgMembers(urlName string) ([]*mono_models.Member, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationMembersParams()
	params.OrganizationName = urlName
	resOk, err := authentication.Client().Organizations.GetOrganizationMembers(params, authentication.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMember fetches the member of an organization accessible to the current user by it's URL Name.
func FetchOrgMember(orgName, name string) (*mono_models.Member, *failures.Failure) {
	members, failure := FetchOrgMembers(orgName)
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

// InviteUserToOrg invites a single user (via email address) to a given
// organization.
//
// The invited user can be added as an owner or a member
//
// Note: This method only returns the invitation for the new user, not existing
// users.
func InviteUserToOrg(org *mono_models.Organization, asOwner bool, email string) (*mono_models.Invitation, *failures.Failure) {
	params := clientOrgs.NewInviteOrganizationParams()
	body := clientOrgs.InviteOrganizationBody{
		AddedOnly: true,
		AsOwner:   asOwner,
	}
	params.SetOrganizationName(org.Urlname)
	params.SetAttributes(body)
	params.SetEmail(email)
	resOk, err := authentication.Client().Organizations.InviteOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, processInviteErrorResponse(err)
	}
	if len(resOk.Payload) != 1 {
		return nil, api.FailUnknown.New("err_api_org_invite_expected_one_invite")
	}
	return resOk.Payload[0], nil

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

func processInviteErrorResponse(err error) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailOrganizationNotFound.New("err_api_org_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
}
