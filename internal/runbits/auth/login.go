package auth

import (
	"errors"
	"net/url"
	"time"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	model "github.com/ActiveState/cli/pkg/platform/model/auth"
	"github.com/go-openapi/strfmt"
)

// OpenURI aliases to osutils.OpenURI which opens the given URI in your browser. This is being exposed so that it can be
// overwritten in tests
var OpenURI = osutils.OpenURI

// Authenticate will prompt the user for authentication
func Authenticate(cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	return AuthenticateWithInput("", "", "", cfg, out, prompt, auth)
}

// AuthenticateWithInput will prompt the user for authentication if the input doesn't already provide it
func AuthenticateWithInput(
	username, password, totp string,
	cfg keypairs.Configurable,
	out output.Outputer,
	prompt prompt.Prompter,
	auth *authentication.Auth,
) error {
	logging.Debug("Authenticating with input")

	credentials := &mono_models.Credentials{Username: username, Password: password, Totp: totp}
	if err := ensureCredentials(credentials, prompt); err != nil {
		return locale.WrapInputError(err, "login_cancelled")
	}

	err := AuthenticateWithCredentials(credentials, auth)
	if err != nil {
		var errTokenRequired *authentication.ErrTokenRequired
		var errUnauthorized *authentication.ErrUnauthorized

		switch {
		case errors.As(err, &errTokenRequired):
			if err := promptToken(credentials, out, prompt, auth); err != nil {
				return errs.Wrap(err, "promptToken failed")
			}
		case errors.As(err, &errUnauthorized):
			return locale.WrapError(err, "err_auth_failed")
		default:
			return locale.WrapError(err, "err_auth_failed_unknown_cause", "", err.Error())
		}
	}

	if auth.Authenticated() {
		secretsapi.InitializeClient(auth)
		if err := ensureUserKeypair(credentials.Password, cfg, out, prompt, auth); err != nil {
			return errs.Wrap(err, "ensureUserKeypair failed")
		}
	}

	return nil
}

// AuthenticateWithToken will try to authenticate with the provided token
func AuthenticateWithToken(token string, auth *authentication.Auth) error {
	logging.Debug("Authenticating with token")

	if err := auth.AuthenticateWithModel(&mono_models.Credentials{
		Token: token,
	}); err != nil {
		return locale.WrapError(err, "err_auth_model", "Failed to authenticate.")
	}

	if err := auth.SaveToken(token); err != nil {
		return locale.WrapError(err, "err_auth_token", "Failed to save token during token authentication.")
	}

	return nil
}

// RequireAuthentication will prompt the user for authentication if they are not already authenticated. If the authentication
// is not successful it will return a failure
func RequireAuthentication(message string, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	if auth.Authenticated() {
		return nil
	}

	out.Print(message)

	choices := []string{
		locale.T("prompt_login_browser_action"),
		locale.T("prompt_login_action"),
		locale.T("prompt_signup_browser_action"),
	}
	choice, err := prompt.Select(locale.Tl("login_signup", "Login or Signup"), locale.T("prompt_login_or_signup"), choices, ptr.To(""), nil)
	if err != nil {
		return errs.Wrap(err, "Prompt cancelled")
	}

	switch choice {
	case locale.T("prompt_login_browser_action"):
		if err := AuthenticateWithBrowser(out, auth, prompt, cfg); err != nil {
			return errs.Wrap(err, "Authenticate failed")
		}
	case locale.T("prompt_login_action"):
		if err := Authenticate(cfg, out, prompt, auth); err != nil {
			return errs.Wrap(err, "Authenticate failed")
		}
	case locale.T("prompt_signup_browser_action"):
		if err := SignupWithBrowser(out, auth, prompt, cfg); err != nil {
			return errs.Wrap(err, "Signup failed")
		}
	}

	if !auth.Authenticated() {
		return locale.NewInputError("err_auth_required")
	}

	return nil
}

func ensureCredentials(credentials *mono_models.Credentials, prompter prompt.Prompter) error {
	var err error
	if credentials.Username == "" {
		if !prompter.IsInteractive() || prompter.IsForced() {
			return locale.NewInputError("err_auth_needinput")
		}
		credentials.Username, err = prompter.Input("", locale.T("username_prompt"), ptr.To(""), nil, prompt.InputRequired)
		if err != nil {
			return errs.Wrap(err, "Input cancelled")
		}
	}

	if credentials.Password == "" {
		if !prompter.IsInteractive() || prompter.IsForced() {
			return locale.NewInputError("err_auth_needinput")
		}
		credentials.Password, err = prompter.InputSecret("", locale.T("password_prompt"), prompt.InputRequired)
		if err != nil {
			return errs.Wrap(err, "Secret input cancelled")
		}
	}
	return nil
}

// AuthenticateWithCredentials will attempt authenticate using the given credentials
func AuthenticateWithCredentials(credentials *mono_models.Credentials, auth *authentication.Auth) error {
	logging.Debug("Authenticating with credentials")

	err := auth.AuthenticateWithModel(credentials)
	if err != nil {
		return err
	}

	if err := auth.CreateToken(); err != nil {
		return locale.WrapError(err, "err_auth_token", "Failed to create token while authenticating with credentials.")
	}

	return nil
}

func promptToken(credentials *mono_models.Credentials, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	var err error
	credentials.Totp, err = prompt.Input("", locale.T("totp_prompt"), ptr.To(""), nil)
	if err != nil {
		return err
	}
	if credentials.Totp == "" {
		out.Notice(locale.T("login_cancelled"))
		return locale.NewInputError("err_auth_empty_token")
	}

	err = AuthenticateWithCredentials(credentials, auth)
	if err != nil {
		return err
	}

	return nil
}

// AuthenticateWithBrowser attempts to authenticate this device with the Platform.
func AuthenticateWithBrowser(out output.Outputer, auth *authentication.Auth, prompt prompt.Prompter, cfg keypairs.Configurable) error {
	logging.Debug("Authenticating with browser")

	err := authenticateWithBrowser(out, auth, prompt, cfg, false)
	if err != nil {
		return errs.Wrap(err, "Error authenticating with browser")
	}

	out.Notice(locale.T("auth_device_success"))

	return nil
}

// authenticateWithBrowser authenticates after signup if applicable.
func authenticateWithBrowser(out output.Outputer, auth *authentication.Auth, prompt prompt.Prompter, cfg keypairs.Configurable, signup bool) error {
	response, err := model.RequestDeviceAuthorization()
	if err != nil {
		return locale.WrapError(err, "err_auth_device")
	}

	if response.VerificationURIComplete == nil {
		return errs.New("Invalid response: Missing verification URL.")
	}

	verificationURL := *response.VerificationURIComplete
	parsedURL, err := url.Parse(verificationURL)
	if err != nil {
		return errs.Wrap(err, "Verification URL is not valid")
	}
	if signup {
		// verificationURL is of the form:
		//   https://platform.activestate.com/authorize/device?user-code=...
		// Transform it to the form:
		//   https://platform.activestate.com/create-account?nextRoute=%2Fauthorize%2Fdevice%3Fuser-code%3D...
		signupURL := api.GetPlatformURL(constants.PlatformSignupPath)
		query := signupURL.Query()
		query.Add("nextRoute", parsedURL.RequestURI())
		signupURL.RawQuery = query.Encode()
		parsedURL = signupURL
	}
	if webclientId := cfg.GetString(anaConst.CfgSessionToken); webclientId != "" {
		query := parsedURL.Query()
		query.Add("webclient_id", webclientId)
		parsedURL.RawQuery = query.Encode()
	}
	verificationURL = parsedURL.String()

	// Print code to user
	if response.UserCode == nil {
		return errs.New("Invalid response: Missing user code.")
	}
	out.Notice(locale.Tr("auth_device_verify_security_code", *response.UserCode, verificationURL))

	// Open URL in browser
	err = OpenURI(verificationURL)
	if err != nil {
		logging.Warning("Could not open browser: %v", err)
		out.Notice(locale.Tr("err_browser_open"))
	}

	var apiKey string
	if !response.Nopoll {
		// Wait for user to complete authentication
		apiKey, err = auth.AuthenticateWithDevicePolling(strfmt.UUID(*response.DeviceCode), time.Duration(response.Interval)*time.Second)
		if err != nil {
			return locale.WrapError(err, "err_auth_device")
		}
	} else {
		// This is the non-default behavior. If Nopoll = true we fall back on prompting the user to continue. It is a
		// failsafe we can use in case polling overloads our API.
		var cont bool
		var err error
		for !cont {
			cont, err = prompt.Confirm(locale.Tl("continue", "Continue?"), locale.T("auth_press_enter"), ptr.To(false), nil)
			if err != nil {
				return errs.Wrap(err, "Not confirmed")
			}
		}
		apiKey, err = auth.AuthenticateWithDevice(strfmt.UUID(*response.DeviceCode))
		if err != nil {
			return locale.WrapError(err, "err_auth_device")
		}
	}

	if err := auth.SaveToken(apiKey); err != nil {
		return locale.WrapError(err, "err_auth_token", "Failed to create token after authenticating with browser.")
	}

	return nil
}
