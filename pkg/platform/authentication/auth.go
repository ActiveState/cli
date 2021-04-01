package authentication

import (
	"errors"
	"fmt"
	"os"

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
}

// Get returns a cached version of Auth
func Get() *Auth {
	if persist == nil {
		cfg, err := config.Get()
		if err != nil {
			// TODO: We need to get rid of this Get() function altogether...
			logging.Error("Could not get configuration required by auth: %v", err)
			os.Exit(1)
		}
		persist = New(cfg)
	}
	return persist
}

// Client is a shortcut for calling Client() on the persisted auth
func Client() *mono_client.Mono {
	return Get().Client()
}

// ClientAuth is a shortcut for calling ClientAuth() on the persisted auth
func ClientAuth() runtime.ClientAuthInfoWriter {
	return Get().ClientAuth()
}

// Reset clears the cache
func Reset() {
	persist = nil
}

// Logout will remove the stored apiToken
func Logout() {
	Get().Logout()
	Reset()
}

// New creates a new version of Auth
func New(cfg Configurable) *Auth {
	auth := &Auth{
		cfg: cfg,
	}

	if availableAPIToken(cfg) != "" {
		logging.Debug("Authenticating with stored API token")
		auth.Authenticate()
	}

	return auth
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
			locale.T("relog_tip", "If you're having trouble authenticating try logging out and logging back in again."),
			locale.T("logout_tip", "Logout with [ACTIONABLE]`state auth logout`[/RESET]."),
			locale.T("logout_tip", "Login with [ACTIONABLE]`state auth`[/RESET]."),
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

// WhoAmI returns the username of the currently authenticated user, or an empty string if not authenticated
func (s *Auth) WhoAmI() string {
	if s.user != nil {
		return s.user.Username
	}
	return ""
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
