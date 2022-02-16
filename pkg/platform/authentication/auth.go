package authentication

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/profile"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/ci/gcloud"
	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"
	apiAuth "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/oauth"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

var exit = os.Exit

var persist *Auth

type ErrUnauthorized struct{ *locale.LocalizedError }

type ErrTokenRequired struct{ *locale.LocalizedError }

// Auth is the base structure used to record the authenticated state
type Auth struct {
	client      *mono_client.Mono
	clientAuth  *runtime.ClientAuthInfoWriter
	bearerToken string
	user        *mono_models.User
	cfg         Configurable
}

type Configurable interface {
	Set(string, interface{}) error
	GetString(string) string
	Close() error
}

const deviceCodeConfigKey = "deviceCode"

// LegacyGet returns a cached version of Auth
func LegacyGet() *Auth {
	if persist == nil {
		cfg, err := config.New()
		if err != nil {
			// TODO: We need to get rid of this Get() function altogether...
			logging.Error("Could not get configuration required by auth: %v", err)
			os.Exit(1)
		}
		persist = New(cfg)
	}
	return persist
}

func LegacyClose() {
	if persist == nil {
		return
	}
	persist.Close()
}

// Client is a shortcut for calling Client() on the persisted auth
func Client() *mono_client.Mono {
	return LegacyGet().Client()
}

// ClientAuth is a shortcut for calling ClientAuth() on the persisted auth
func ClientAuth() runtime.ClientAuthInfoWriter {
	return LegacyGet().ClientAuth()
}

// Reset clears the cache
func Reset() {
	persist = nil
}

// Logout will remove the stored apiToken
func Logout() {
	LegacyGet().Logout()
	Reset()
}

// New creates a new version of Auth
func New(cfg Configurable) *Auth {
	defer profile.Measure("auth:New", time.Now())
	auth := &Auth{
		cfg: cfg,
	}

	if availableAPIToken(cfg) != "" || cfg.GetString(deviceCodeConfigKey) != "" {
		logging.Debug("Authenticating with stored API token or device code")
		auth.Authenticate()
	}

	return auth
}

func (s *Auth) Close() error {
	if err := s.cfg.Close(); err != nil {
		return errs.Wrap(err, "Could not close cfg from Auth")
	}
	return nil
}

func (s *Auth) storeAuthenticatedDevice(deviceCode strfmt.UUID, response *oauth.AuthDeviceGetOK) error {
	defer s.updateRollbarPerson()

	s.user = response.Payload.AccessToken.User
	s.bearerToken = response.Payload.AccessToken.Token
	clientAuth := httptransport.BearerToken(s.bearerToken)
	s.clientAuth = &clientAuth

	err := s.cfg.Set(deviceCodeConfigKey, deviceCode)
	if err != nil {
		return errs.Wrap(err, "Could not set deviceCode credentials in config")
	}

	return err
}

// Authenticated checks whether we are currently authenticated
func (s *Auth) Authenticated() bool {
	if s.clientAuth != nil {
		return true
	}
	existingDeviceCode := s.cfg.GetString(deviceCodeConfigKey)
	if existingDeviceCode == "" || s.bearerToken != "" {
		return false
	}
	// Check if the device is still authenticated with the Platform and if so, get a token.
	deviceCode := strfmt.UUID(existingDeviceCode)
	params := oauth.NewAuthDeviceGetParams()
	params.SetDeviceCode(deviceCode)
	if response, err := mono.Get().Oauth.AuthDeviceGet(params); response != nil {
		if err := s.storeAuthenticatedDevice(deviceCode, response); err == nil {
			return true
		}
		if err != nil {
			logging.Error("Error storing authenticated device", err.Error())
		}
	} else {
		// Either the token for deviceCode has expired, we are rate-limited by the Platform and
		// have to try again later, or the Platform is unreachable.
		// Rate limiting can happen during testing.
		if badRequest, ok := err.(*oauth.AuthDeviceGetBadRequest); ok && *badRequest.Payload.Error == oauth.AuthDeviceGetBadRequestBodyErrorSlowDown {
			logging.Warning("Attempting to query the Platform for device authentication status too frequently.")
		}
	}
	return false
}

// ClientAuth returns the auth type required by swagger api calls
func (s *Auth) ClientAuth() runtime.ClientAuthInfoWriter {
	if s.clientAuth == nil {
		return nil
	}
	return *s.clientAuth
}

// BearerToken returns the current bearerToken
func (s *Auth) BearerToken() string {
	return s.bearerToken
}

func (s *Auth) updateRollbarPerson() {
	uid := s.UserID()
	if uid == nil {
		return
	}
	logging.UpdateRollbarPerson(uid.String(), s.WhoAmI(), s.Email())
}

// Authenticate will try to authenticate using stored credentials
func (s *Auth) Authenticate() error {
	if s.Authenticated() {
		s.updateRollbarPerson()
		return nil
	}

	apiToken := availableAPIToken(s.cfg)
	if apiToken == "" {
		return locale.NewInputError("err_no_credentials")
	}

	return s.AuthenticateWithToken(apiToken)
}

// AuthenticateWithModel will try to authenticate using the given swagger model
func (s *Auth) AuthenticateWithModel(credentials *mono_models.Credentials) error {
	params := authentication.NewPostLoginParams()
	params.SetCredentials(credentials)

	loginOK, err := mono.Get().Authentication.PostLogin(params)
	if err != nil {
		tips := []string{
			locale.Tl("relog_tip", "If you're having trouble authenticating try logging out and logging back in again."),
			locale.Tl("logout_tip", "Logout with [ACTIONABLE]`state auth logout`[/RESET]."),
			locale.Tl("logout_tip", "Login with [ACTIONABLE]`state auth`[/RESET]."),
		}

		switch err.(type) {
		case *apiAuth.PostLoginUnauthorized:
			return errs.AddTips(&ErrUnauthorized{locale.WrapInputError(err, "err_unauthorized")}, tips...)
		case *apiAuth.PostLoginRetryWith:
			return errs.AddTips(&ErrTokenRequired{locale.WrapInputError(err, "err_auth_fail_totp")}, tips...)
		default:
			logging.Error("Authentication API returned %v", err)
			return errs.AddTips(locale.WrapError(err, "err_api_auth", "Authentication failed: {{.V0}}", err.Error()), tips...)
		}
	}
	defer s.updateRollbarPerson()

	payload := loginOK.Payload
	s.user = payload.User
	s.bearerToken = payload.Token
	clientAuth := httptransport.BearerToken(s.bearerToken)
	s.clientAuth = &clientAuth

	if credentials.Token != "" {
		setErr := s.cfg.Set("apiToken", credentials.Token)
		if setErr != nil {
			return errs.Wrap(err, "Could not set API token credentials in config")
		}
	} else {
		if err := s.CreateToken(); err != nil {
			return errs.Wrap(err, "CreateToken failed")
		}
	}

	return nil
}

// AuthenticateWithUser will try to authenticate using the given credentials
func (s *Auth) AuthenticateWithUser(username, password, totp string) error {
	return s.AuthenticateWithModel(&mono_models.Credentials{
		Username: username,
		Password: password,
		Totp:     totp,
	})
}

// AuthenticateWithToken will try to authenticate using the given token
func (s *Auth) AuthenticateWithToken(token string) error {
	return s.AuthenticateWithModel(&mono_models.Credentials{
		Token: token,
	})
}

// AuthenticateWithDeviceCode posts a request to authenticate this device on the Platform and waits
// for the user to authorize the request.
// The given callback function is called when the user should be prompted to authorize the request.
// Returns an error if authentication cannot be performed (e.g. timeout or Platform is unreachable).
func (s *Auth) AuthenticateWithDevice(promptCallback func(userCode, uri string)) error {
	if s.Authenticated() {
		return nil // nothing to do
	}
	// Post the authentication request to the Platform.
	postParams := oauth.NewAuthDevicePostParams()
	response, err := mono.Get().Oauth.AuthDevicePost(postParams)
	if err != nil {
		logging.Error("Error requesting device authentication: %v", err)
		return locale.NewError("err_auth_device")
	}
	// Prompt the user to authorize the request.
	promptCallback(*response.Payload.UserCode, *response.Payload.VerificationURIComplete)
	// Wait for the authorization.
	deviceCode := strfmt.UUID(*response.Payload.DeviceCode)
	getParams := oauth.NewAuthDeviceGetParams()
	getParams.SetDeviceCode(deviceCode)
	startTime := time.Now()
	const timeout = 6 * 60 * time.Second
	for {
		response, err := mono.Get().Oauth.AuthDeviceGet(getParams)
		if response != nil {
			err := s.storeAuthenticatedDevice(deviceCode, response)
			if err != nil {
				logging.Error("Error storing authenticated device", err.Error())
			}
			break
		} else if errs.Matches(err, &oauth.AuthDeviceGetBadRequest{}) {
			badRequest := err.(*oauth.AuthDeviceGetBadRequest)
			errorString := *badRequest.Payload.Error
			if errorString == oauth.AuthDeviceGetBadRequestBodyErrorExpiredToken || time.Since(startTime) >= timeout {
				return locale.NewInputError("auth_device_timeout")
			} else if errorString == oauth.AuthDeviceGetBadRequestBodyErrorInvalidClient {
				logging.Error("Error requesting device authentication: invalid client") // IP address mismatch
				return locale.NewError("err_auth_device")
			} else if errorString == oauth.AuthDeviceGetBadRequestBodyErrorSlowDown {
				logging.Warning("Attempting to check for authorization status too frequently.")
			}
			time.Sleep(5 * time.Second) // then try again
		} else {
			logging.Error("Error requesting device authentication status: %v", err)
			return locale.NewError("err_auth_device")
		}
	}
	return nil
}

// WhoAmI returns the username of the currently authenticated user, or an empty string if not authenticated
func (s *Auth) WhoAmI() string {
	if s.user != nil {
		return s.user.Username
	}
	return ""
}

func (s *Auth) CanWrite(organization string) bool {
	if s.user == nil {
		return false
	}
	for _, org := range s.user.Organizations {
		if org.URLname != organization {
			continue
		}
		return org.Role == string(mono_models.RoleAdmin) || org.Role == string(mono_models.RoleEditor)
	}
	return false
}

// Email return the email of the authenticated user
func (s *Auth) Email() string {
	if s.user != nil {
		return s.user.Email
	}
	return ""
}

// UserID returns the user ID for the currently authenticated user, or nil if not authenticated
func (s *Auth) UserID() *strfmt.UUID {
	if s.user != nil {
		return &s.user.UserID
	}
	return nil
}

// Logout will destroy any session tokens and reset the current Auth instance
func (s *Auth) Logout() {
	err := s.cfg.Set("apiToken", "")
	if err != nil {
		logging.Error("Could not clear apiToken in config")
	}
	err = s.cfg.Set(deviceCodeConfigKey, "")
	if err != nil {
		logging.Error("Could not clear deviceCode key in config")
	}
	s.client = nil
	s.clientAuth = nil
	s.bearerToken = ""
	s.user = nil
}

// Client will return an API client that has authentication set up
func (s *Auth) Client() *mono_client.Mono {
	client, err := s.ClientSafe()
	if err != nil {
		logging.Error("Trying to get the Client while not authenticated")
		fmt.Fprintln(os.Stderr, colorize.StripColorCodes(locale.T("err_api_not_authenticated")))
		exit(1)
	}

	return client
}

// ClientSafe will return an API client that has authentication set up
func (s *Auth) ClientSafe() (*mono_client.Mono, error) {
	if s.client == nil {
		s.client = mono.NewWithAuth(s.clientAuth)
	}
	if !s.Authenticated() {
		if err := s.Authenticate(); err != nil {
			return nil, errs.Wrap(err, "Authentication failed")
		}
	}
	return s.client, nil
}

// CreateToken will create an API token for the current authenticated user
func (s *Auth) CreateToken() error {
	client, err := s.ClientSafe()
	if err != nil {
		return err
	}

	tokensOK, err := client.Authentication.ListTokens(nil, s.ClientAuth())
	if err != nil {
		return locale.WrapError(err, "err_token_list", "", err.Error())
	}

	for _, token := range tokensOK.Payload {
		if token.Name == constants.APITokenName {
			params := authentication.NewDeleteTokenParams()
			params.SetTokenID(token.TokenID)
			_, err := client.Authentication.DeleteToken(params, s.ClientAuth())
			if err != nil {
				return locale.WrapError(err, "err_token_delete", "", err.Error())
			}
			break
		}
	}

	key := constants.APITokenName + ":" + machineid.UniqID()
	token, err := s.NewAPIKey(key)
	if err != nil {
		return err
	}

	err = s.cfg.Set("apiToken", token)
	if err != nil {
		return locale.WrapError(err, "err_set_token", "Could not set token in config")
	}

	return nil
}

// NewAPIKey returns a new api key from the backend or the relevant failure.
func (s *Auth) NewAPIKey(name string) (string, error) {
	params := authentication.NewAddTokenParams()
	params.SetTokenOptions(&mono_models.TokenEditable{Name: name})

	client, err := s.ClientSafe()
	if err != nil {
		return "", err
	}

	tokenOK, err := client.Authentication.AddToken(params, s.ClientAuth())
	if err != nil {
		return "", locale.WrapError(err, "err_token_create", "", err.Error())
	}

	return tokenOK.Payload.Token, nil
}

func availableAPIToken(cfg Configurable) string {
	tkn, err := gcloud.GetSecret(constants.APIKeyEnvVarName)
	if err != nil && !errors.Is(err, gcloud.ErrNotAvailable{}) {
		logging.Error("Could not retrieve gcloud secret: %v", err)
	}
	if err == nil && tkn != "" {
		logging.Debug("Using api token sourced from gcloud")
		return tkn
	}

	if tkn = os.Getenv(constants.APIKeyEnvVarName); tkn != "" {
		logging.Debug("Using API token passed via env var")
		return tkn
	}
	return cfg.GetString("apiToken")
}
