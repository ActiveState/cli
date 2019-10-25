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
)

// Authenticate will prompt the user for authentication
func Authenticate() {
	AuthenticateWithInput("", "")
}

// AuthenticateWithInput will prompt the user for authentication if the input doesn't already provide it
func AuthenticateWithInput(username string, password string) {
	logging.Debug("AuthenticateWithInput")
	credentials := &mono_models.Credentials{Username: username, Password: password}
	if err := promptForLogin(credentials); err != nil {
		failures.Handle(err, locale.T("err_prompt_unkown"))
		return
	}

	fail := AuthenticateWithCredentials(credentials)
	if fail != nil {
		switch fail.Type {
		case authentication.FailAuthUnauthorized:
			if !uniqueUsername(credentials) {
				failures.Handle(fail, locale.T("err_auth_failed"))
				return
			}
			promptSignup(credentials)
		case authentication.FailAuthNeedToken:
			promptToken(credentials)
		default:
			failures.Handle(fail, locale.T("err_auth_failed_unknown_cause"))
		}
	}

	if authentication.Get().Authenticated() {
		secretsapi.InitializeClient()
		ensureUserKeypair(credentials.Password)
	}

	// ensure changes are propagated
	config.Save()
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

	print.Line(locale.T("login_success_welcome_back", map[string]string{
		"Name": auth.WhoAmI(),
	}))

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

func promptSignup(credentials *mono_models.Credentials) {
	yesSignup, fail := Prompter.Confirm(locale.T("prompt_login_to_signup"), true)
	if fail != nil {
		failures.Handle(fail, locale.T("err_auth_signup_failed"))
		return
	}
	if yesSignup {
		signupFromLogin(credentials.Username, credentials.Password)
	}
}

func promptToken(credentials *mono_models.Credentials) {
	var fail *failures.Failure
	credentials.Totp, fail = Prompter.Input(locale.T("totp_prompt"), "")
	if fail != nil {
		failures.Handle(fail, locale.T("err_auth_fail_totp"))
		return
	}
	if credentials.Totp == "" {
		print.Line(locale.T("login_cancelled"))
		return
	}

	fail = AuthenticateWithCredentials(credentials)
	if fail != nil {
		failures.Handle(fail, locale.T("err_ath_failed_unknown_cause"))
	}
}
