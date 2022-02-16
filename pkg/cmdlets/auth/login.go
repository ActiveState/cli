package auth

import (
	"github.com/skratchdot/open-golang/open"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// OpenURI aliases to open.Run which opens the given URI in your browser. This is being exposed so that it can be
// overwritten in tests
var OpenURI = open.Run

// Authenticate will prompt the user for authentication
func Authenticate(cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter) error {
	return AuthenticateWithInput("", "", "", cfg, out, prompt)
}

// AuthenticateWithInput will prompt the user for authentication if the input doesn't already provide it
func AuthenticateWithInput(username, password, totp string, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter) error {
	logging.Debug("AuthenticateWithInput")
	credentials := &mono_models.Credentials{Username: username, Password: password, Totp: totp}
	if err := promptForLogin(credentials, prompt); err != nil {
		return locale.WrapInputError(err, "login_cancelled")
	}

	err := AuthenticateWithCredentials(credentials)
	if err != nil {
		switch {
		case errs.Matches(err, &authentication.ErrTokenRequired{}):
			if err := promptToken(credentials, out, prompt); err != nil {
				return errs.Wrap(err, "promptToken failed")
			}
		case errs.Matches(err, &authentication.ErrUnauthorized{}):
			if !uniqueUsername(credentials) {
				return errs.Wrap(err, "uniqueUsername failed")
			}
			if err := promptSignup(credentials, out, prompt); err != nil {
				return errs.Wrap(err, "promptSignup failed")
			}
		default:
			return locale.WrapError(err, "err_auth_failed_unknown_cause", "", err.Error())
		}
	}

	if authentication.LegacyGet().Authenticated() {
		secretsapi.InitializeClient()
		if err := ensureUserKeypair(credentials.Password, cfg, out, prompt); err != nil {
			return errs.Wrap(err, "ensureUserKeypair failed")
		}
	}

	return nil
}

// RequireAuthentication will prompt the user for authentication if they are not already authenticated. If the authentication
// is not successful it will return a failure
func RequireAuthentication(message string, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter) error {
	if authentication.LegacyGet().Authenticated() {
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
		if err := AuthenticateWithDevice(out); err != nil {
			return errs.Wrap(err, "Authenticate failed")
		}
	case locale.T("prompt_login_action"):
		if err := Authenticate(cfg, out, prompt); err != nil {
			return errs.Wrap(err, "Authenticate failed")
		}
	case locale.T("prompt_signup_browser_action"):
		if err := AuthenticateWithDevice(out); err != nil { // user can sign up from this page too
			return errs.Wrap(err, "Signup failed")
		}
	case locale.T("prompt_signup_action"):
		if err := Signup(cfg, out, prompt); err != nil {
			return errs.Wrap(err, "Signup failed")
		}
	}

	if !authentication.LegacyGet().Authenticated() {
		return locale.NewInputError("err_auth_required")
	}

	return nil
}

func promptForLogin(credentials *mono_models.Credentials, prompter prompt.Prompter) error {
	var err error
	if credentials.Username == "" {
		credentials.Username, err = prompter.Input("", locale.T("username_prompt"), new(string), prompt.InputRequired)
		if err != nil {
			return errs.Wrap(err, "Input cancelled")
		}
	}

	if credentials.Password == "" {
		credentials.Password, err = prompter.InputSecret("", locale.T("password_prompt"), prompt.InputRequired)
		if err != nil {
			return errs.Wrap(err, "Secret input cancelled")
		}
	}
	return nil
}

// AuthenticateWithCredentials will attempt authenticate using the given credentials
func AuthenticateWithCredentials(credentials *mono_models.Credentials) error {
	auth := authentication.LegacyGet()
	err := auth.AuthenticateWithModel(credentials)
	if err != nil {
		return err
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
		return false
	}

	return true
}

func promptSignup(credentials *mono_models.Credentials, out output.Outputer, prompt prompt.Prompter) error {
	loginConfirmDefault := true
	yesSignup, err := prompt.Confirm("", locale.T("prompt_login_to_signup"), &loginConfirmDefault)
	if err != nil {
		return err
	}
	if yesSignup {
		return signupFromLogin(credentials.Username, credentials.Password, out, prompt)
	}

	return nil
}

func promptToken(credentials *mono_models.Credentials, out output.Outputer, prompt prompt.Prompter) error {
	var err error
	credentials.Totp, err = prompt.Input("", locale.T("totp_prompt"), new(string))
	if err != nil {
		return err
	}
	if credentials.Totp == "" {
		out.Notice(locale.T("login_cancelled"))
		return locale.NewInputError("err_auth_empty_token")
	}

	err = AuthenticateWithCredentials(credentials)
	if err != nil {
		return err
	}

	return nil
}

// AuthenticateWithDevice attempts to authenticate this device with the Platform.
func AuthenticateWithDevice(out output.Outputer) error {
	deviceCode, err := model.RequestDeviceAuthorization()
	if err != nil {
		return err
	}
	out.Notice(locale.Tr("auth_device_verify_security_code", *deviceCode.UserCode))
	err = OpenURI(*deviceCode.VerificationURIComplete)
	if err != nil {
		logging.Error("Could not open browser: %v", err)
		out.Notice(locale.Tr("err_browser_open", *deviceCode.VerificationURIComplete))
	}
	authorization, err := model.WaitForAuthorization(deviceCode)
	if err != nil {
		return err
	}
	err = authentication.LegacyGet().AuthenticateWithDevice(deviceCode, authorization.AccessToken)
	if err != nil {
		return err
	}
	out.Notice(locale.T("auth_device_success"))
	return nil
}
