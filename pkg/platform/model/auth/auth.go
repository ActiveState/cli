package model

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/oauth"
	mms "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

// RequestDeviceAuthorization posts a request to authorize this device on the Platform and
// returns the device code needed for authorization.
// The user is subsequently required to visit the device code's URI and click the "Authorize"
// button.
func RequestDeviceAuthorization() (*mms.DeviceCode, error) {
	postParams := oauth.NewAuthDevicePostParams()
	response, err := mono.Get().Oauth.AuthDevicePost(postParams)
	if err != nil {
		return nil, errs.Wrap(err, "Could not request device authentication")
	}

	return response.Payload, nil
}

func CheckDeviceAuthorization(deviceCode strfmt.UUID) (jwt *mms.JWT, apiKey *mms.NewToken, err error) {
	getParams := oauth.NewAuthDeviceGetParams()
	getParams.SetDeviceCode(deviceCode)

	response, err := mono.Get().Oauth.AuthDeviceGet(getParams)
	if err != nil {
		var errAuthDeviceGetBadRequest *oauth.AuthDeviceGetBadRequest

		// Identify input or benign errors
		if errors.As(err, &errAuthDeviceGetBadRequest) {
			errorToken := errAuthDeviceGetBadRequest.Payload.Error
			switch *errorToken {
			case oauth.AuthDeviceGetBadRequestBodyErrorAuthorizationPending, oauth.AuthDeviceGetBadRequestBodyErrorSlowDown:
				logging.Debug("Authorization still pending")
				return nil, nil, nil
			case oauth.AuthDeviceGetBadRequestBodyErrorExpiredToken:
				return nil, nil, locale.WrapExternalError(err, "auth_device_timeout")
			}
		}

		return nil, nil, errs.Wrap(err, api.ErrorMessageFromPayload(err))
	}

	return response.Payload.AccessToken, response.Payload.RefreshToken, nil
}
