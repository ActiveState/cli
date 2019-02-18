package auth

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/surveyor"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/client/authentication"
	"github.com/ActiveState/cli/pkg/platform/api/client/users"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func plainAuth() {
	credentials := &models.Credentials{}
	if err := promptForLogin(credentials); err != nil {
		failures.Handle(err, locale.T("err_prompt_unkown"))
		return
	}

	doPlainAuth(credentials)

	if api.Auth != nil {
		secretsapi.InitializeClient()
		ensureUserKeypair(credentials.Password)
	}
}

func promptForLogin(credentials *models.Credentials) error {
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

	return survey.Ask(qs, credentials)
}

func doPlainAuth(credentials *models.Credentials) {
	loginOK, err := api.Authenticate(credentials)

	// Error checking
	if err != nil {
		switch err.(type) {
		// Authentication failed due to username not existing
		case *authentication.PostLoginUnauthorized:
			params := users.NewUniqueUsernameParams()
			params.SetUsername(credentials.Username)
			_, err := api.Client.Users.UniqueUsername(params)
			if err == nil {
				if promptConfirm("prompt_login_to_signup") {
					signupFromLogin(credentials.Username, credentials.Password)
				}
			} else {
				failures.Handle(err, locale.T("err_auth_failed"))
			}
			return
		case *authentication.PostLoginRetryWith:
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
			doPlainAuth(credentials)
			return
		default:
			failures.Handle(err, locale.T("err_auth_failed_unknown_cause"))
			return
		}
	}

	print.Line(locale.T("login_success_welcome_back", map[string]string{
		"Name": loginOK.Payload.User.Username,
	}))
}
