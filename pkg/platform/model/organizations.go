package model

import (
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	clientOrgs "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FailOrgResponseLen is a failure due to the response length not matching the request length
var FailOrgResponseLen = failures.Type("model.fail.getcommithistory")

// FetchOrganizations fetches all organizations for the current user.
func FetchOrganizations() ([]*mono_models.Organization, *failures.Failure) {
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
func FetchOrgByURLName(urlName string) (*mono_models.Organization, *failures.Failure) {
	params := clientOrgs.NewGetOrganizationParams()
	params.OrganizationIdentifier = urlName
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
	authClient, fail := authentication.Get().ClientSafe()
	if fail != nil {
		return nil, fail
	}
	resOk, err := authClient.Organizations.GetOrganizationMembers(params, authentication.ClientAuth())
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
func InviteUserToOrg(orgName string, asOwner bool, email string) (*mono_models.Invitation, *failures.Failure) {
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
		return nil, api.FailUnknown.New("err_api_org_invite_expected_one_invite")
	}
	return resOk.Payload[0], nil

}

// FetchOrganizationsByIDs fetches organizations by their IDs
func FetchOrganizationsByIDs(ids []strfmt.UUID) ([]model.Organization, *failures.Failure) {
	ids = funk.Uniq(ids).([]strfmt.UUID)
	request := request.OrganizationsByIDs(ids)

	gql := graphql.Get()
	response := model.Organizations{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, api.FailUnknown.Wrap(err)
	}

	if len(response.Organizations) != len(ids) {
		return nil, FailOrgResponseLen.New(locale.Tr("err_orgs_length"))
	}

	return response.Organizations, nil
}

// FetchOrganizationByID fetches an organization by its ID
func FetchOrganizationByID(id strfmt.UUID) (*mono_models.Organization, error) {
	params := clientOrgs.NewGetOrganizationParams()
	params.SetOrganizationIdentifier(id.String())
	idType := "organizationID"
	params.SetIdentifierType(&idType)

	res, err := authentication.Client().Organizations.GetOrganization(params, authentication.ClientAuth())
	if err != nil {
		return nil, locale.WrapError(err, "err_fetch_org_by_id", "Could not find organization with ID: {{.V0}}", id.String())
	}
	return res.Payload, nil
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
	case 400:
		return api.FailUnknown.Wrap(locale.WrapInputError(err, "err_api_invite_400", "Invalid request, did you enter a valid email address?"))
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailOrganizationNotFound.New("err_api_org_not_found")
	default:
		return api.FailUnknown.Wrap(errs.New(api.ErrorMessageFromPayload(err)))
	}
}
