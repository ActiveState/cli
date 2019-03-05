package auth

import (
	"errors"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/surveyor"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/client/users"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

type signupInput struct {
	Name      string
	Email     string
	Username  string
	Password  string
	Password2 string
}

// Signup will prompt the user to create an account
func Signup() {
	input := &signupInput{}

	err := promptForSignup(input)
	if err != nil {
		failures.Handle(err, locale.T("err_prompt_unkown"))
		return
	}

	doSignup(input)

	if authentication.Get().Authenticated() {
		if failure := generateKeypairForUser(input.Password); failure != nil {
			failures.Handle(failure, locale.T("keypair_err_save"))
		}
	}
}

func signupFromLogin(username string, password string) {
	input := &signupInput{}

	input.Username = username
	input.Password = password
	err := promptForSignup(input)
	if err != nil {
		failures.Handle(err, locale.T("err_prompt_unkown"))
		return
	}

	doSignup(input)
}

func promptForSignup(input *signupInput) error {
	qs := []*survey.Question{}

	if input.Username != "" {
		print.Line(locale.T("confirm_password_account_creation"))
	} else {
		qs = append(qs, []*survey.Question{
			{
				Name:     "username",
				Prompt:   &survey.Input{Message: locale.T("username_prompt_signup")},
				Validate: survey.ComposeValidators(surveyor.ValidateRequired, UsernameValidator),
			},
			{
				Name:     "password",
				Prompt:   &survey.Password{Message: locale.T("password_prompt_signup")},
				Validate: surveyor.ValidateRequired,
			},
		}...)
	}

	// Must define password validator here as it has to reference the input
	var passwordValidator = func(val interface{}) error {
		value := val.(string)
		if value != input.Password {
			return errors.New(locale.T("err_password_confirmation_failed"))
		}
		return nil
	}

	qs = append(qs, []*survey.Question{
		{
			Name:     "password2",
			Prompt:   &survey.Password{Message: locale.T("password_prompt_confirm")},
			Validate: passwordValidator,
		},
		{
			Name:     "name",
			Prompt:   &survey.Input{Message: locale.T("name_prompt")},
			Validate: surveyor.ValidateRequired,
		},
		{
			Name:     "email",
			Prompt:   &survey.Input{Message: locale.T("email_prompt")},
			Validate: surveyor.ValidateRequired,
		},
	}...)

	err := survey.Ask(qs, input)
	if err != nil {
		return err
	}
	return nil
}

func doSignup(input *signupInput) {
	params := users.NewAddUserParams()
	params.SetUser(&models.UserEditable{
		Email:    input.Email,
		Username: input.Username,
		Password: input.Password,
		Name:     input.Name,
	})
	addUserOK, err := api.Get().Users.AddUser(params)

	// Error checking
	if err != nil {
		switch err.(type) {
		// Authentication failed due to email already existing (username check already happened at this point)
		case *users.AddUserConflict:
			failures.Handle(err, locale.T("err_auth_signup_email_exists"))
		default:
			failures.Handle(err, locale.T("err_auth_failed_unknown_cause"))
		}
		return
	}

	AuthenticateWithCredentials(&models.Credentials{
		Username: input.Username,
		Password: input.Password,
	})

	print.Line(locale.T("signup_success", map[string]string{
		"Email": addUserOK.Payload.User.Email,
	}))
}

// UsernameValidator verifies whether the chosen username is usable
func UsernameValidator(val interface{}) error {
	value := val.(string)
	params := users.NewUniqueUsernameParams()
	params.SetUsername(value)
	res, err := api.Get().Users.UniqueUsername(params)
	if err != nil || *res.Payload.Code != int64(200) {
		return errors.New(locale.T("err_username_taken"))
	}
	return nil
}
