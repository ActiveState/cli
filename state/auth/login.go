package auth

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/client/authentication"
	"github.com/ActiveState/cli/internal/api/client/users"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/surveyor"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

var testCredentials *models.Credentials

func plainAuth() {
	credentials := &models.Credentials{}

	if testCredentials != nil {
		credentials = testCredentials
	}

	err := promptForLogin(credentials)
	if err != nil {
		failures.Handle(err, locale.T("err_prompt_unkown"))
		return
	}

	doPlainAuth(credentials)
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

	err := survey.Ask(qs, credentials)
	if err != nil {
		return err
	}

	return nil
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
			res, err := api.Client.Users.UniqueUsername(params)
			if err == nil && *res.Payload.Code == int64(200) {
				confirmed := false
				prompt := &survey.Confirm{
					Message: locale.T("prompt_login_to_signup"),
				}
				survey.AskOne(prompt, &confirmed, nil)
				if confirmed {
					signupFromLogin(credentials.Username, credentials.Password)
				}
			} else {
				print.Error(locale.T("err_auth_failed"))
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
