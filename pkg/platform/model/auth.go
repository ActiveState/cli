package model

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/oauth"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// RequestDeviceAuthorization posts a request to authorize this device on the Platform and
// returns the device code needed for authorization.
// The user is subsequently required to visit the device code's URI and click the "Authorize"
// button.
func RequestDeviceAuthorization() (*mono_models.DeviceCode, error) {
	postParams := oauth.NewAuthDevicePostParams()
	response, err := mono.Get().Oauth.AuthDevicePost(postParams)
	if err != nil {
		logging.Error("Error requesting device authorization: %v", err)
		return nil, locale.NewError("err_auth_device")
	}
	return response.Payload, nil
}

// WaitForAuthorization waits for the user to authorize a previously posted device authorization
// request and returns the completed authorization.
// Returns an error if authorization cannot be performed (e.g. timeout or Platform is unreachable).
func WaitForAuthorization(deviceCodePayload *mono_models.DeviceCode) (*mono_models.DeviceCodeComplete, error) {
	deviceCode := strfmt.UUID(*deviceCodePayload.DeviceCode)
	getParams := oauth.NewAuthDeviceGetParams()
	getParams.SetDeviceCode(deviceCode)
	startTime := time.Now()
	const timeout = 5 * 60 * time.Second
	for {
		response, err := mono.Get().Oauth.AuthDeviceGet(getParams)
		switch {
		case response != nil:
			return response.Payload, nil
		case errs.Matches(err, &oauth.AuthDeviceGetBadRequest{}):
			badRequest := err.(*oauth.AuthDeviceGetBadRequest)
			errorToken := *badRequest.Payload.Error
			if errorToken == oauth.AuthDeviceGetBadRequestBodyErrorExpiredToken || time.Since(startTime) >= timeout {
				return nil, locale.NewInputError("auth_device_timeout")
			} else if errorToken == oauth.AuthDeviceGetBadRequestBodyErrorInvalidClient {
				logging.Error("Error requesting device authentication: invalid client") // IP address mismatch
				return nil, locale.NewError("err_auth_device")
			} else if errorToken == oauth.AuthDeviceGetBadRequestBodyErrorSlowDown {
				logging.Warning("Attempting to check for authorization status too frequently.")
			}
			time.Sleep(time.Duration(deviceCodePayload.Interval) * time.Second) // then try again
		default:
			return nil, locale.WrapError(err, "err_auth_device")
		}
	}
}
