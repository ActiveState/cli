package model

import (
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	clientOrgs "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var ErrMemberNotFound = errs.New("member not found")

// FetchOrganizations fetches all organizations for the current user.
func FetchOrganizations(auth *authentication.Auth) ([]*mono_models.Organization, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	params := clientOrgs.NewListOrganizationsParams()
	memberOnly := true
	params.SetMemberOnly(&memberOnly)
	res, err := authClient.Organizations.ListOrganizations(params, auth.ClientAuth())

	if err != nil {
		return nil, processOrgErrorResponse(err)
	}

	return res.Payload, nil
}

// FetchOrgByURLName fetches an organization accessible to the current user by it's URL Name.
func FetchOrgByURLName(urlName string, auth *authentication.Auth) (*mono_models.Organization, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationIdentifier = urlName
	resOk, err := authClient.Organizations.GetOrganization(params, auth.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMembers fetches the members of an organization accessible to the current user by it's URL Name.
func FetchOrgMembers(urlName string, auth *authentication.Auth) ([]*mono_models.Member, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	params := clientOrgs.NewGetOrganizationMembersParams()
	params.OrganizationName = urlName
	resOk, err := authClient.Organizations.GetOrganizationMembers(params, auth.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMember fetches the member of an organization accessible to the current user by it's URL Name.
func FetchOrgMember(orgName, name string, auth *authentication.Auth) (*mono_models.Member, error) {
	members, err := FetchOrgMembers(orgName, auth)
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		if strings.EqualFold(name, member.User.Username) {
			return member, nil
		}
	}

	return nil, locale.WrapError(ErrMemberNotFound, "err_api_member_not_found")
}

// InviteUserToOrg invites a single user (via email address) to a given
// organization.
//
// # The invited user can be added as an owner or a member
//
// Note: This method only returns the invitation for the new user, not existing
// users.
func InviteUserToOrg(orgName string, asOwner bool, email string, auth *authentication.Auth) (*mono_models.Invitation, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	params := clientOrgs.NewInviteOrganizationParams()
	role := mono_models.RoleReader
	if asOwner {
		role = mono_models.RoleAdmin
	}
	body := clientOrgs.InviteOrganizationBody{
		AddedOnly: true,
		Role:      &role,
	}
	params.SetOrganizationName(orgName)
	params.SetAttributes(body)
	params.SetEmail(email)
	resOk, err := authClient.Organizations.InviteOrganization(params, auth.ClientAuth())
	if err != nil {
		return nil, processInviteErrorResponse(err)
	}
	if len(resOk.Payload) != 1 {
		return nil, locale.NewError("err_api_org_invite_expected_one_invite")
	}
	return resOk.Payload[0], nil

}

// FetchOrganizationsByIDs fetches organizations by their IDs
func FetchOrganizationsByIDs(ids []strfmt.UUID, auth *authentication.Auth) ([]model.Organization, error) {
	ids = funk.Uniq(ids).([]strfmt.UUID)
	request := request.OrganizationsByIDs(ids)

	gql := graphql.New(auth)
	response := model.Organizations{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, errs.Wrap(err, "gql.Run failed")
	}

	if len(response.Organizations) != len(ids) {
		logging.Debug("Organization membership mismatch: %d members returned for %d members requested. Caller must account for this.", len(response.Organizations), len(ids))
	}

	return response.Organizations, nil
}

func processOrgErrorResponse(err error) error {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return locale.NewExternalError("err_api_not_authenticated")
	case 404:
		return locale.NewExternalError("err_api_org_not_found")
	default:
		return err
	}
}

func processInviteErrorResponse(err error) error {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 400:
		return locale.WrapExternalError(err, "err_api_invite_400", "Invalid request. Did you enter a valid email address?")
	case 401:
		return locale.NewExternalError("err_api_not_authenticated")
	case 404:
		return locale.NewExternalError("err_api_org_not_found")
	default:
		return locale.WrapError(err, api.ErrorMessageFromPayload(err))
	}
}
