package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Role string

const (
	// RoleAdmin represents an organizaiton role that has admin rights
	RoleAdmin = Role(mono_models.RoleAdmin)
	// RoleEditor represents an organizaiton role that has editor rights
	RoleEditor = Role(mono_models.RoleEditor)
	// RoleReader represents an organizaiton role that has reader rights
	RoleReader = Role(mono_models.RoleReader)
)

type ErrNotMember struct{ *locale.LocalizedError }

func UserOrgRole(username, orgName string) (Role, error) {
	params := users.NewGetUserParams()
	params.SetUsername(username)

	resOk, err := authentication.Client().Users.GetUser(params, authentication.ClientAuth())
	if err != nil {
		return "", locale.WrapError(err, "err_get_user_org_role", "Could not get user's role in organization: {{.V0}}", api.ErrorMessageFromPayload(err))
	}

	for _, org := range resOk.Payload.Organizations {
		if org.URLname != orgName {
			continue
		}
		return Role(org.Role), nil
	}

	return "", &ErrNotMember{locale.NewError("err_orgs_no_user", "User: {{.V0}} is not a member of the organization: {{.V1}}", username, orgName)}
}

func UserOrgEditPermission(username, orgName string) (bool, error) {
	role, err := UserOrgRole(username, orgName)
	if err != nil {
		if errs.Matches(err, &ErrNotMember{}) {
			return false, nil
		}
		return false, locale.WrapError(err, "err_user_org_role", "Could not determine user's role in organization")
	}

	return role == RoleAdmin || role == RoleEditor, nil
}
