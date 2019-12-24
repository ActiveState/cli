package auth

import (
	"github.com/skratchdot/open-golang/open"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// OpenURI aliases to open.Run which opens the given URI in your browser. This is being exposed so that it can be
// overwritten in tests
var OpenURI = open.Run

var (
	// FailLoginPrompt indicates a failure during the login prompt
	FailLoginPrompt = failures.Type("auth.fail.loginprompt", failures.FailUserInput)

	// FailNotAuthenticated conveys a failure to authenticate by the user
	FailNotAuthenticated = failures.Type("auth.fail.notauthenticated", failures.FailUserInput)

	// FailBrowserOpen indicates a failure to open the users browser
	FailBrowserOpen = failures.Type("auth.failure.browseropen")

	// FailAuthUnknown conveys a failure to authenticated with unknown cause
	FailAuthUnknown = failures.Type("auth.failure.unknown")

	// FailEmptyToken indicates the token provided by the user was empty
	FailEmptyToken = failures.Type("auth.failure.emptytoken")
)

// Authenticate will prompt the user for authentication
func Authenticate() {
	AuthenticateWithInput("", "")
}

// AuthenticateWithInput will prompt the user for authentication if the input doesn't already provide it
func AuthenticateWithInput(username string, password string) *failures.Failure {
	logging.Debug("AuthenticateWithInput")
	credentials := &mono_models.Credentials{Username: username, Password: password}
	if err := promptForLogin(credentials); err != nil {
		return failures.FailUserInput.Wrap(err)
	}

	fail := AuthenticateWithCredentials(credentials)
	if fail != nil {
		switch fail.Type {
		case authentication.FailAuthUnauthorized:
			if !uniqueUsername(credentials) {
				return fail
			}
			fail = promptSignup(credentials)
			if fail != nil {
				return fail
			}
		case authentication.FailAuthNeedToken:
			fail = promptToken(credentials)
			if fail != nil {
				return fail
			}
		default:
			return FailAuthUnknown.New(locale.T("err_auth_failed_unknown_cause"))
		}
	}

	if authentication.Get().Authenticated() {
		secretsapi.InitializeClient()
		fail = ensureUserKeypair(credentials.Password)
		if fail != nil {
			return fail
		}
	}

	// ensure changes are propagated
	err := config.Save()
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	return nil
}

// RequireAuthentication will prompt the user for authentication if they are not already authenticated. If the authentication
// is not succesful it will return a failure
func RequireAuthentication(message string) *failures.Failure {
	if authentication.Get().Authenticated() {
		return nil
	}

	print.Info(message)

	choices := []string{locale.T("prompt_login_action"), locale.T("prompt_signup_action"), locale.T("prompt_signup_browser_action")}
	choice, fail := Prompter.Select(locale.T("prompt_login_or_signup"), choices, "")
	if fail != nil {
		return fail
	}

	switch choice {
	case locale.T("prompt_login_action"):
		Authenticate()
	case locale.T("prompt_signup_action"):
		Signup()
	case locale.T("prompt_signup_browser_action"):
		err := OpenURI(constants.PlatformSignupURL)
		if err != nil {
			logging.Error("Could not open browser: %v", err)
			return FailBrowserOpen.New(locale.Tr("err_browser_open", constants.PlatformSignupURL))
		}
		print.Info(locale.T("prompt_login_after_browser_signup"))
		Authenticate()
	}

	if !authentication.Get().Authenticated() {
		return FailNotAuthenticated.New(locale.T("err_auth_required"))
	}

	return nil
}

func promptForLogin(credentials *mono_models.Credentials) *failures.Failure {
	var fail *failures.Failure
	if credentials.Username == "" {
		credentials.Username, fail = Prompter.Input(locale.T("username_prompt"), "", prompt.InputRequired)
		if fail != nil {
			return FailLoginPrompt.Wrap(fail.ToError())
		}
	}

	if credentials.Password == "" {
		credentials.Password, fail = Prompter.InputSecret(locale.T("password_prompt"), prompt.InputRequired)
		if fail != nil {
			return FailLoginPrompt.Wrap(fail.ToError())
		}
	}
	return nil
}

// AuthenticateWithCredentials will attempt authenticate using the given credentials
func AuthenticateWithCredentials(credentials *mono_models.Credentials) *failures.Failure {
	auth := authentication.Get()
	fail := auth.AuthenticateWithModel(credentials)
	if fail != nil {
		return fail
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

func promptSignup(credentials *mono_models.Credentials) *failures.Failure {
	yesSignup, fail := Prompter.Confirm(locale.T("prompt_login_to_signup"), true)
	if fail != nil {
		return fail
	}
	if yesSignup {
		return signupFromLogin(credentials.Username, credentials.Password)
	}

	return nil
}

func promptToken(credentials *mono_models.Credentials) *failures.Failure {
	var fail *failures.Failure
	credentials.Totp, fail = Prompter.Input(locale.T("totp_prompt"), "")
	if fail != nil {
		return fail
	}
	if credentials.Totp == "" {
		print.Line(locale.T("login_cancelled"))
		return FailEmptyToken.New(locale.T("err_auth_empty_token"))
	}

	fail = AuthenticateWithCredentials(credentials)
	if fail != nil {
		return fail
	}

	return nil
}
