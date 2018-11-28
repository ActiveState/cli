package api

import (
	"flag"
	"fmt"
	"net/http"
	"reflect"

	"github.com/ActiveState/cli/internal/api/client"
	"github.com/ActiveState/cli/internal/api/client/authentication"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/spf13/viper"
)

// Client contains the active API Client connection
var Client *client.APIClient

// BearerToken holds the user's Bearer-token received from the API
var BearerToken string

// Auth holds our authenticated information, go-swagger makes us pass this manually to all calls that require auth
var Auth runtime.ClientAuthInfoWriter

// Prefix is the URL prefix for our API, intended for use in tests
var Prefix string

var (
	// FailUnknown is the failure type used for API requests with an unexpected error
	FailUnknown = failures.Type("api.fail.unknown")

	// FailAuth is the failure type used for failed authentication API requests
	FailAuth = failures.Type("api.fail.auth", failures.FailUser)

	// FailNotFound indicates a failure to find a user's resource.
	FailNotFound = failures.Type("api.fail.not_found", failures.FailUser)

	// FailOrganizationNotFound is used when a project could not be found
	FailOrganizationNotFound = failures.Type("api.fail.organization.not_found", FailNotFound)

	// FailProjectNotFound is used when a project could not be found
	FailProjectNotFound = failures.Type("api.fail.project.not_found", FailNotFound)
)

var transport http.RoundTripper

func init() {
	ReInitialize()
}

// ReInitialize initializes (or re-initializes) an API connection
func ReInitialize() {
	transportRuntime := httptransport.New(constants.APIHost, constants.APIPath, []string{constants.APISchema})
	if flag.Lookup("test.v") != nil {
		transportRuntime.SetDebug(true)
	}
	Prefix = fmt.Sprintf("%s://%s%s", constants.APISchema, constants.APIHost, constants.APIPath)

	if flag.Lookup("test.v") != nil {
		transportRuntime.Transport = transport
	}
	if BearerToken != "" {
		Auth = httptransport.BearerToken(BearerToken)
		transportRuntime.DefaultAuthentication = Auth
	}
	Client = client.New(transportRuntime, strfmt.Default)

	apiToken := viper.GetString("apiToken")
	if BearerToken == "" && apiToken != "" {
		_, err := Authenticate(&models.Credentials{
			Token: apiToken,
		})
		if err != nil {
			logging.Warningf("Authentication failed: %s", err.Error())
			viper.Set("apiToken", "")
		}
	}
}

// Authenticate authenticates us against the API
func Authenticate(credentials *models.Credentials) (*authentication.PostLoginOK, error) {
	logging.Debug("Authenticate")

	params := authentication.NewPostLoginParams()
	params.SetCredentials(credentials)
	loginOK, err := Client.Authentication.PostLogin(params)

	if err != nil {
		return nil, err
	}

	// NOTE (gus) there's a chance for an infinite loop if BearerToken is not set for some reason
	BearerToken = loginOK.Payload.Token
	ReInitialize()

	if credentials.Token != "" {
		viper.Set("apiToken", credentials.Token)
	} else {
		persistWithToken()
	}

	return loginOK, nil
}

// RemoveAuth removes any authentication info stored and reinitializes our API connection
func RemoveAuth() {
	viper.Set("apiToken", "")
	BearerToken = ""
	Auth = nil
	ReInitialize()
}

// ErrorCode tries to retrieve the code associated with an API error
func ErrorCode(err interface{}) int {
	codeVal := reflect.Indirect(reflect.ValueOf(err)).FieldByName("Code")
	if codeVal.IsValid() {
		return int(codeVal.Int())
	}
	return ErrorCodeFromPayload(err)
}

// ErrorCodeFromPayload tries to retrieve the code associated with an API error from a
// Message object referenced as a Payload.
func ErrorCodeFromPayload(err interface{}) int {
	errVal := reflect.ValueOf(err)
	payloadVal := reflect.Indirect(errVal).FieldByName("Payload")
	if !payloadVal.IsValid() {
		return -1
	}

	codePtr := reflect.Indirect(payloadVal).FieldByName("Code")
	if !codePtr.IsValid() {
		return -1
	}

	codeVal := reflect.Indirect(codePtr)
	if !codeVal.IsValid() {
		return -1
	}
	return int(codeVal.Int())
}

// persistWithToken will retrieve and save a persistent authentication token based on the active authentication information
func persistWithToken() {
	logging.Debug("Persisting token")

	tokensOK, err := Client.Authentication.ListTokens(nil, Auth)
	if err != nil {
		logging.Errorf("Something went wrong whilst trying to retrieve tokens: %s", err.Error())
		return
	}

	for _, token := range tokensOK.Payload {
		if token.Name == constants.APITokenName {
			params := authentication.NewDeleteTokenParams()
			params.SetTokenID(token.TokenID)
			_, err := Client.Authentication.DeleteToken(params, Auth)
			if err != nil {
				logging.Errorf("Could not delete old token: %s", err.Error())
				return
			}
			break
		}
	}

	params := authentication.NewAddTokenParams()
	params.SetTokenOptions(&models.TokenEditable{Name: constants.APITokenName})
	tokenOK, err := Client.Authentication.AddToken(params, Auth)
	if err != nil {
		logging.Errorf("Could not create new token: %s", err.Error())
		return
	}

	token := tokenOK.Payload.Token
	logging.Debug("Value: %s", token)
	viper.Set("apiToken", token)
}
