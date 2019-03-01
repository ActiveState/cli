package auth

import (
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/surveyor"
	"github.com/ActiveState/cli/pkg/platform/api"
	apiAuth "github.com/ActiveState/cli/pkg/platform/api/client/authentication"
	"github.com/ActiveState/cli/pkg/platform/api/client/users"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

var (
	// FailLoginPrompt indicates a failure during the login prompt
	FailLoginPrompt = failures.Type("auth.fail.loginprompt", failures.FailUserInput)
)

// Authenticate will prompt the user for authentication
func Authenticate() {
	credentials := &models.Credentials{}
	if err := promptForLogin(credentials); err != nil {
		failures.Handle(err, locale.T("err_prompt_unkown"))
		return
	}

	AuthenticateWithCredentials(credentials)

	if authentication.Get().Authenticated() {
		secretsapi.InitializeClient()
		ensureUserKeypair(credentials.Password)
	}
}

func promptForLogin(credentials *models.Credentials) *failures.Failure {
	var qs = []*survey.Question{
		{
			Name:     "username",
			Prompt:   &survey.Input{Message: locale.T("username_prompt")},
			Validate: surveyor.ValidateRequired,
		},
		{
			Name:     "password",
			Prompt:   &survey.Password{Message: locale.T("password_prompt")},
			Validate: surveyor.ValidateRequired,
		},
	}

	err := survey.Ask(qs, credentials)
	if err != nil {
		return FailLoginPrompt.Wrap(err)
	}
	return nil
}

// AuthenticateWithCredentials wil lauthenticate using the given credentials, it's main purpose is to communicate
// any failures to the end-user
func AuthenticateWithCredentials(credentials *models.Credentials) {
	auth := authentication.Get()
	fail := auth.AuthenticateWithModel(credentials)

	// Error checking
	if fail != nil {
		switch fail.ToError().(type) {
		// Authentication failed due to username not existing
		case *apiAuth.PostLoginUnauthorized:
			params := users.NewUniqueUsernameParams()
			params.SetUsername(credentials.Username)
			_, err := api.Get().Users.UniqueUsername(params)
			if err == nil {
				if promptConfirm("prompt_login_to_signup") {
					signupFromLogin(credentials.Username, credentials.Password)
				}
			} else {
				failures.Handle(err, locale.T("err_auth_failed"))
			}
			return
		case *apiAuth.PostLoginRetryWith:
			var qs = []*survey.Question{
				{
					Name:     "totp",
					Prompt:   &survey.Input{Message: locale.T("totp_prompt")},
					Validate: surveyor.ValidateRequired,
				},
			}
			survey.Ask(qs, credentials)
			if credentials.Totp == "" {
				print.Line(locale.T("login_cancelled"))
				return
			}
			AuthenticateWithCredentials(credentials)
			return
		default:
			failures.Handle(fail, locale.T("err_auth_failed_unknown_cause"))
			return
		}
	}

	print.Line(locale.T("login_success_welcome_back", map[string]string{
		"Name": auth.WhoAmI(),
	}))
}
