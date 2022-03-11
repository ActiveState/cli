package auth

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model/auth"
	"github.com/go-openapi/strfmt"
	"github.com/skratchdot/open-golang/open"
)

// OpenURI aliases to open.Run which opens the given URI in your browser. This is being exposed so that it can be
// overwritten in tests
var OpenURI = open.Run

// Authenticate will prompt the user for authentication
func Authenticate(cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	return AuthenticateWithInput("", "", "", false, cfg, out, prompt, auth)
}

// AuthenticateWithInput will prompt the user for authentication if the input doesn't already provide it
func AuthenticateWithInput(username, password, totp string, nonInteractive bool, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	logging.Debug("Authenticating with input")

	credentials := &mono_models.Credentials{Username: username, Password: password, Totp: totp}
	if err := ensureCredentials(credentials, prompt, nonInteractive); err != nil {
		return locale.WrapInputError(err, "login_cancelled")
	}

	err := AuthenticateWithCredentials(credentials, auth)
	if err != nil {
		switch {
		case errs.Matches(err, &authentication.ErrTokenRequired{}):
			if err := promptToken(credentials, out, prompt, auth); err != nil {
				return errs.Wrap(err, "promptToken failed")
			}
		case errs.Matches(err, &authentication.ErrUnauthorized{}):
			if !uniqueUsername(credentials) {
				return errs.Wrap(err, "uniqueUsername failed")
			}
			if err := promptSignup(credentials, out, prompt, auth); err != nil {
				return errs.Wrap(err, "promptSignup failed")
			}
		default:
			return locale.WrapError(err, "err_auth_failed_unknown_cause", "", err.Error())
		}
	}

	if auth.Authenticated() {
		secretsapi.InitializeClient()
		if err := ensureUserKeypair(credentials.Password, cfg, out, prompt); err != nil {
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
		return locale.WrapError(err, "err_auth_token")
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
		locale.T("prompt_signup_action"),
	}
	choice, err := prompt.Select(locale.Tl("login_signup", "Login or Signup"), locale.T("prompt_login_or_signup"), choices, new(string))
	if err != nil {
		return errs.Wrap(err, "Prompt cancelled")
	}

	switch choice {
	case locale.T("prompt_login_browser_action"):
		if err := AuthenticateWithBrowser(out, auth, prompt); err != nil {
			return errs.Wrap(err, "Authenticate failed")
		}
	case locale.T("prompt_login_action"):
		if err := Authenticate(cfg, out, prompt, auth); err != nil {
			return errs.Wrap(err, "Authenticate failed")
		}
	case locale.T("prompt_signup_browser_action"):
		if err := AuthenticateWithBrowser(out, auth, prompt); err != nil { // user can sign up from this page too
			return errs.Wrap(err, "Signup failed")
		}
	case locale.T("prompt_signup_action"):
		if err := Signup(cfg, out, prompt, auth); err != nil {
			return errs.Wrap(err, "Signup failed")
		}
	}

	if !auth.Authenticated() {
		return locale.NewInputError("err_auth_required")
	}

	return nil
}

func ensureCredentials(credentials *mono_models.Credentials, prompter prompt.Prompter, nonInteractive bool) error {
	var err error
	if credentials.Username == "" {
		if nonInteractive {
			return locale.NewInputError("err_auth_needinput")
		}
		credentials.Username, err = prompter.Input("", locale.T("username_prompt"), new(string), prompt.InputRequired)
		if err != nil {
			return errs.Wrap(err, "Input cancelled")
		}
	}

	if credentials.Password == "" {
		if nonInteractive {
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
		return locale.WrapError(err, "err_auth_token")
	}

	return nil
}

func uniqueUsername(credentials *mono_models.Credentials) bool {
	params := users.NewUniqueUsernameParams()
	params.SetUsername(credentials.Username)
	_, err := mono.Get().Users.UniqueUsername(params)
	if err != nil {
		// This error is not useful to the user so we do not return it and log instead
		logging.Error("Error when checking for unique username: %v", err)
		rollbar.Error("Error when checking for unique username: %v", err)
		return false
	}

	return true
}

func promptSignup(credentials *mono_models.Credentials, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	loginConfirmDefault := true
	yesSignup, err := prompt.Confirm("", locale.T("prompt_login_to_signup"), &loginConfirmDefault)
	if err != nil {
		return err
	}
	if yesSignup {
		return signupFromLogin(credentials.Username, credentials.Password, out, prompt, auth)
	}

	return nil
}

func promptToken(credentials *mono_models.Credentials, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	var err error
	credentials.Totp, err = prompt.Input("", locale.T("totp_prompt"), new(string))
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
func AuthenticateWithBrowser(out output.Outputer, auth *authentication.Auth, prompt prompt.Prompter) error {
	logging.Debug("Authenticating with browser")

	response, err := model.RequestDeviceAuthorization()
	if err != nil {
		return locale.WrapError(err, "err_auth_device")
	}

	// Print code to user
	if response.UserCode == nil {
		return errs.New("Invalid response: Missing user code.")
	}
	out.Notice(locale.Tr("auth_device_verify_security_code", *response.UserCode))

	// Open URL in browser
	if response.VerificationURIComplete == nil {
		return errs.New("Invalid response: Missing verification URL.")
	}
	err = OpenURI(*response.VerificationURIComplete)
	if err != nil {
		logging.Warning("Could not open browser: %v", err)
		out.Notice(locale.Tr("err_browser_open", *response.VerificationURIComplete))
	}

	if !response.Nopoll {
		// Wait for user to complete authentication
		if err := auth.AuthenticateWithDevicePolling(strfmt.UUID(*response.DeviceCode), time.Duration(response.Interval)*time.Second); err != nil {
			return locale.WrapError(err, "err_auth_device")
		}
	} else {
		// This is the non-default behavior. If Nopoll = true we fall back on prompting the user to continue. It is a
		// failsafe we can use in case polling overloads our API.
		var cont bool
		var err error
		for !cont {
			cont, err = prompt.Confirm(locale.Tl("continue", "Continue?"), locale.T("auth_press_enter"), p.BoolP(false))
			if err != nil {
				return errs.Wrap(err, "Prompt failed")
			}
		}
		if err := auth.AuthenticateWithDevice(strfmt.UUID(*response.DeviceCode)); err != nil {
			return locale.WrapError(err, "err_auth_device")
		}
	}

	if err := auth.CreateToken(); err != nil {
		return locale.WrapError(err, "err_auth_token")
	}

	out.Notice(locale.T("auth_device_success"))

	return nil
}
