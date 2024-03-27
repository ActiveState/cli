package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientLimits "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/limits"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchOrganizationLimits returns the limits for an organization
func FetchOrganizationLimits(orgName string, auth *authentication.Auth) (*mono_models.Limits, error) {
	params := clientLimits.NewGetOrganizationLimitsParams()
	params.SetOrganizationIdentifier(orgName)
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	res, err := authClient.Limits.GetOrganizationLimits(params, auth.ClientAuth())

	if err != nil {
		return nil, processLimitsErrorResponse(err)
	}

	return res.Payload, nil
}

func processLimitsErrorResponse(err error) error {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return locale.NewError("err_api_not_authenticated")
	case 403:
		return locale.NewError("err_api_forbidden")
	case 404:
		return locale.NewError("err_api_org_not_found")
	default:
		return errs.Wrap(err, "Unknown failure")
	}
}
