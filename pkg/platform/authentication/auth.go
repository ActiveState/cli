package authentication

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/ci/gcloud"
	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"
	apiAuth "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model/auth"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

var exit = os.Exit

var persist *Auth

type ErrUnauthorized struct{ *locale.LocalizedError }

type ErrTokenRequired struct{ *locale.LocalizedError }

var errNotYetGranted = locale.NewInputError("err_auth_device_noauth")

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

const ApiTokenConfigKey = "apiToken"

// LegacyGet returns a cached version of Auth
func LegacyGet() *Auth {
	if persist == nil {
		cfg, err := config.New()
		if err != nil {
			// TODO: We need to get rid of this Get() function altogether...
			multilog.Error("Could not get configuration required by auth: %v", err)
			os.Exit(1)
		}
		defer cfg.Close()
		
		persist = New(cfg)
		if err := persist.Sync(); err != nil {
			logging.Warning("Could not sync authenticated state: %s", err.Error())
		}
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

// New creates a new version of Auth
func New(cfg Configurable) *Auth {
	defer profile.Measure("auth:New", time.Now())
	auth := &Auth{
		cfg: cfg,
	}

	return auth
}

// Sync will ensure that the authenticated state is in sync with what is in the config database.
// This is mainly useful if you want to instrument the auth package without creating unnecessary API calls.
func (s *Auth) Sync() error {
	defer profile.Measure("auth:Sync", time.Now())

	if s.AvailableAPIToken() != "" {
		logging.Debug("Authenticating with stored API token")
		if err := s.Authenticate(); err != nil {
			return errs.Wrap(err, "Failed to authenticate with API token")
		}
	}
	return nil
}

func (s *Auth) Close() error {
	if err := s.cfg.Close(); err != nil {
		return errs.Wrap(err, "Could not close cfg from Auth")
	}
	return nil
}

// Authenticated checks whether we are currently authenticated
func (s *Auth) Authenticated() bool {
	return s.clientAuth != nil
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
	rollbar.UpdateRollbarPerson(uid.String(), s.WhoAmI(), s.Email())
}

// Authenticate will try to authenticate using stored credentials
func (s *Auth) Authenticate() error {
	if s.Authenticated() {
		s.updateRollbarPerson()
		return nil
	}

	apiToken := s.AvailableAPIToken()
	if apiToken == "" {
		return locale.NewInputError("err_no_credentials")
	}

	return s.AuthenticateWithToken(apiToken)
}

// AuthenticateWithModel will try to authenticate using the given swagger model
func (s *Auth) AuthenticateWithModel(credentials *mono_models.Credentials) error {
	logging.Debug("AuthenticateWithModel")

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
			multilog.Error("Authentication API returned %v", err)
			return errs.AddTips(locale.WrapError(err, "err_api_auth", "Authentication failed: {{.V0}}", err.Error()), tips...)
		}
	}

	if err := s.updateSession(loginOK.Payload); err != nil {
		return errs.Wrap(err, "Storing JWT failed")
	}

	return nil
}

func (s *Auth) AuthenticateWithDevice(deviceCode strfmt.UUID) error {
	logging.Debug("AuthenticateWithDevice")

	token, err := model.CheckDeviceAuthorization(deviceCode)
	if err != nil {
		return errs.Wrap(err, "Authorization failed")
	}

	if token == nil {
		return errNotYetGranted
	}

	if err := s.updateSession(token); err != nil {
		return errs.Wrap(err, "Storing JWT failed")
	}

	return nil

}

func (s *Auth) AuthenticateWithDevicePolling(deviceCode strfmt.UUID, interval time.Duration) error {
	logging.Debug("AuthenticateWithDevicePolling, polling: %v", interval.String())
	for start := time.Now(); time.Since(start) < 5*time.Minute; {
		err := s.AuthenticateWithDevice(deviceCode)
		if err == nil {
			return nil
		} else if !errors.Is(err, errNotYetGranted) {
			return errs.Wrap(err, "Device authentication failed")
		}
		time.Sleep(interval) // then try again
	}

	return locale.NewInputError("err_auth_device_timeout")
}

// AuthenticateWithToken will try to authenticate using the given token
func (s *Auth) AuthenticateWithToken(token string) error {
	logging.Debug("AuthenticateWithToken")
	return s.AuthenticateWithModel(&mono_models.Credentials{
		Token: token,
	})
}

// updateSession authenticates with the given access token obtained via a Platform
// API request and response (e.g. username/password loging or device authentication).
func (s *Auth) updateSession(accessToken *mono_models.JWT) error {
	defer s.updateRollbarPerson()

	s.user = accessToken.User
	s.bearerToken = accessToken.Token
	clientAuth := httptransport.BearerToken(s.bearerToken)
	s.clientAuth = &clientAuth

	persist = s

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
func (s *Auth) Logout() error {
	err := s.cfg.Set(ApiTokenConfigKey, "")
	if err != nil {
		multilog.Error("Could not clear apiToken in config")
		return locale.WrapError(err, "err_logout_cfg", "Could not update config, if this persists please try running '[ACTIONABLE]state clean config[/RESET]'.")
	}

	s.client = nil
	s.clientAuth = nil
	s.bearerToken = ""
	s.user = nil

	// This is a bit of a hack, but it's safe to assume that the global legacy use-case should be reset whenever we logout a specific instance
	// Handling it any other way would be far too error-prone by comparison
	Reset()

	return nil
}

// Client will return an API client that has authentication set up
func (s *Auth) Client() *mono_client.Mono {
	client, err := s.ClientSafe()
	if err != nil {
		multilog.Error("Trying to get the Client while not authenticated")
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

	key := constants.APITokenName + ":" + uniqid.Text()
	token, err := s.NewAPIKey(key)
	if err != nil {
		return err
	}

	err = s.SaveToken(token)
	if err != nil {
		return errs.Wrap(err, "SaveToken failed")
	}

	return nil
}

// SaveToken will save an API token
func (s *Auth) SaveToken(token string) error {
	err := s.cfg.Set(ApiTokenConfigKey, token)
	if err != nil {
		return locale.WrapError(err, "err_set_token", "Could not set token in config")
	}

	return nil
}

// NewAPIKey returns a new api key from the backend or the relevant failure.
func (s *Auth) NewAPIKey(name string) (string, error) {
	params := authentication.NewAddTokenParams()
	params.SetTokenOptions(&mono_models.TokenEditable{Name: name, DeviceID: uniqid.Text()})

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

func (s *Auth) AvailableAPIToken() (v string) {
	tkn, err := gcloud.GetSecret(constants.APIKeyEnvVarName)
	if err != nil && !errors.Is(err, gcloud.ErrNotAvailable{}) {
		multilog.Error("Could not retrieve gcloud secret: %v", err)
	}
	if err == nil && tkn != "" {
		logging.Debug("Using api token sourced from gcloud")
		return tkn
	}

	if tkn = os.Getenv(constants.APIKeyEnvVarName); tkn != "" {
		logging.Debug("Using API token passed via env var")
		return tkn
	}
	return s.cfg.GetString(ApiTokenConfigKey)
}
