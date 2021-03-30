package model

import (
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
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
func FetchOrganizations() ([]*mono_models.Organization, error) {
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
func FetchOrgByURLName(urlName string) (*mono_models.Organization, error) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationIdentifier = urlName
	resOk, err := authentication.Client().Organizations.GetOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMembers fetches the members of an organization accessible to the current user by it's URL Name.
func FetchOrgMembers(urlName string) ([]*mono_models.Member, error) {
	params := clientOrgs.NewGetOrganizationMembersParams()
	params.OrganizationName = urlName
	authClient, err := authentication.Get().ClientSafe()
	if err != nil {
		return nil, err
	}
	resOk, err := authClient.Organizations.GetOrganizationMembers(params, authentication.ClientAuth())
	if err != nil {
		return nil, processOrgErrorResponse(err)
	}
	return resOk.Payload, nil
}

// FetchOrgMember fetches the member of an organization accessible to the current user by it's URL Name.
func FetchOrgMember(orgName, name string) (*mono_models.Member, error) {
	members, err := FetchOrgMembers(orgName)
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
// The invited user can be added as an owner or a member
//
// Note: This method only returns the invitation for the new user, not existing
// users.
func InviteUserToOrg(orgName string, asOwner bool, email string) (*mono_models.Invitation, error) {
	params := clientOrgs.NewInviteOrganizationParams()
	body := clientOrgs.InviteOrganizationBody{
		AddedOnly: true,
		AsOwner:   asOwner,
	}
	params.SetOrganizationName(orgName)
	params.SetAttributes(body)
	params.SetEmail(email)
	resOk, err := authentication.Client().Organizations.InviteOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, processInviteErrorResponse(err)
	}
	if len(resOk.Payload) != 1 {
		return nil, locale.NewError("err_api_org_invite_expected_one_invite")
	}
	return resOk.Payload[0], nil

}

// FetchOrganizationsByIDs fetches organizations by their IDs
func FetchOrganizationsByIDs(ids []strfmt.UUID) ([]model.Organization, error) {
	ids = funk.Uniq(ids).([]strfmt.UUID)
	request := request.OrganizationsByIDs(ids)

	gql := graphql.New()
	response := model.Organizations{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, errs.Wrap(err, "gql.Run failed")
	}

	if len(response.Organizations) != len(ids) {
		return nil, locale.NewError("err_orgs_length")
	}

	return response.Organizations, nil
}

func processOrgErrorResponse(err error) error {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return locale.NewInputError("err_api_not_authenticated")
	case 404:
		return locale.NewInputError("err_api_org_not_found")
	default:
		return err
	}
}

func processInviteErrorResponse(err error) error {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 400:
		return locale.WrapInputError(err, "err_api_invite_400", "Invalid request, did you enter a valid email address?")
	case 401:
		return locale.NewInputError("err_api_not_authenticated")
	case 404:
		return locale.NewInputError("err_api_org_not_found")
	default:
		return locale.WrapError(err, api.ErrorMessageFromPayload(err))
	}
}
