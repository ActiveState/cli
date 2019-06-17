package auth

import (
	"errors"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
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
	var fail *failures.Failure

	if input.Username != "" {
		print.Line(locale.T("confirm_password_account_creation"))
	} else {
		input.Username, fail = Prompter.Input(locale.T("username_prompt_signup"), "", prompt.InputRequired)
		if fail != nil {
			return fail.ToError()
		}
		input.Password, fail = Prompter.InputSecret(locale.T("password_prompt_signup"), prompt.InputRequired)
		if fail != nil {
			return fail.ToError()
		}
	}

	// Must define password validator here as it has to reference the input
	var passwordValidator = func(val interface{}) error {
		value := val.(string)
		if value != input.Password {
			return errors.New(locale.T("err_password_confirmation_failed"))
		}
		return nil
	}

	input.Password2, fail = Prompter.InputSecret(locale.T("password_prompt_confirm"), prompt.InputRequired)
	if fail != nil {
		return fail.ToError()
	}
	err := passwordValidator(input.Password2)
	if err != nil {
		return err
	}

	input.Name, fail = Prompter.Input(locale.T("name_prompt"), "", prompt.InputRequired)
	if fail != nil {
		return fail.ToError()
	}

	input.Email, fail = Prompter.Input(locale.T("email_prompt"), "", prompt.InputRequired)
	if fail != nil {
		return fail.ToError()
	}
	return nil
}

func doSignup(input *signupInput) {
	params := users.NewAddUserParams()
	params.SetUser(&mono_models.UserEditable{
		Email:    input.Email,
		Username: input.Username,
		Password: input.Password,
		Name:     input.Name,
	})
	addUserOK, err := mono.Get().Users.AddUser(params)

	// Error checking
	if err != nil {
		errMsg := api.ErrorMessageFromPayload(err)
		if errMsg == "" {
			errMsg = err.Error()
		}
		switch err.(type) {
		// Authentication failed due to email already existing (username check already happened at this point)
		case *users.AddUserConflict:
			failures.Handle(errors.New(errMsg), locale.T("err_auth_signup_email_exists"))
		default:
			failures.Handle(errors.New(errMsg), locale.T("err_auth_failed_unknown_cause"))
		}
		return
	}

	AuthenticateWithCredentials(&mono_models.Credentials{
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
	res, err := mono.Get().Users.UniqueUsername(params)
	if err != nil || *res.Payload.Code != int64(200) {
		return errors.New(locale.T("err_username_taken"))
	}
	return nil
}
