package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientLimits "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/limits"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchOrganizationLimits returns the limits for an organization
func FetchOrganizationLimits(orgName string) (*mono_models.Limits, *failures.Failure) {
	params := clientLimits.NewGetOrganizationLimitsParams()
	params.SetOrganizationName(orgName)
	res, err := authentication.Client().Limits.GetOrganizationLimits(params, authentication.ClientAuth())

	if err != nil {
		return nil, processLimitsErrorResponse(err)
	}

	return res.Payload, nil
}

func processLimitsErrorResponse(err error) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 403:
		return api.FailForbidden.New("err_api_forbidden")
	case 404:
		return api.FailOrganizationNotFound.New("err_api_org_not_found")
	default:
		return api.FailUnknown.Wrap(err)
	}
}
