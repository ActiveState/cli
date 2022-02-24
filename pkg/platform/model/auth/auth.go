package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/oauth"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

// RequestDeviceAuthorization posts a request to authorize this device on the Platform and
// returns the device code needed for authorization.
// The user is subsequently required to visit the device code's URI and click the "Authorize"
// button.
func RequestDeviceAuthorization() (*mono_models.DeviceCode, error) {
	postParams := oauth.NewAuthDevicePostParams()
	response, err := mono.Get().Oauth.AuthDevicePost(postParams)
	if err != nil {
		return nil, errs.Wrap(err, "Could not request device authentication")
	}

	return response.Payload, nil
}

func CheckDeviceAuthorization(deviceCode strfmt.UUID) (*mono_models.JWT, error) {
	getParams := oauth.NewAuthDeviceGetParams()
	getParams.SetDeviceCode(deviceCode)

	response, err := mono.Get().Oauth.AuthDeviceGet(getParams)
	if err != nil {
		// Identify input or benign errors
		if errs.Matches(err, &oauth.AuthDeviceGetBadRequest{}) {
			errorToken := err.(*oauth.AuthDeviceGetBadRequest).Payload.Error
			switch *errorToken {
			case oauth.AuthDeviceGetBadRequestBodyErrorAuthorizationPending, oauth.AuthDeviceGetBadRequestBodyErrorSlowDown:
				return nil, nil
			case oauth.AuthDeviceGetBadRequestBodyErrorExpiredToken:
				return nil, locale.WrapInputError(err, "auth_device_timeout")
			}
		}

		return nil, errs.Wrap(err, api.ErrorMessageFromPayload(err))
	}

	return response.Payload.AccessToken, nil
}
