package auth

import (
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/multilog"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/prompt"
	"github.com/ActiveState/cli/internal/keypairs"

	"github.com/ActiveState/cli/pkg/cmdlets/legalprompt"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

const (
	maxMatchTries = 3
)

type signupInput struct {
	Email     string
	Username  string
	Password  string
	Password2 string
}

// Signup will prompt the user to create an account
func Signup(cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	accepted, err := legalprompt.TOS(out, prompt)
	if err != nil {
		return err
	}
	if !accepted {
		return locale.NewInputError("tos_not_accepted", "")
	}

	input := &signupInput{}
	err = promptForSignup(input, maxMatchTries, out, prompt)
	if err != nil {
		return locale.WrapError(err, "signup_failure")
	}

	if err = doSignup(input, out, auth); err != nil {
		return err
	}

	if auth.Authenticated() {
		if err := generateKeypairForUser(cfg, input.Password); err != nil {
			return locale.WrapError(err, "keypair_err_generate")
		}
	}

	return nil
}

func signupFromLogin(username string, password string, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	input := &signupInput{}

	input.Username = username
	input.Password = password

	err := promptForSignup(input, maxMatchTries, out, prompt)
	if err != nil {
		return locale.WrapError(err, "signup_failure")
	}

	return doSignup(input, out, auth)
}

func promptForSignup(input *signupInput, matchTries int, out output.Outputer, prompter prompt.Prompter) error {
	var err error

	if input.Username != "" {
		out.Notice(locale.T("confirm_password_account_creation"))
	} else {
		input.Username, err = prompter.Input("", locale.T("username_prompt_signup"), new(string), prompt.InputRequired)
		if err != nil {
			return err
		}
		input.Password, err = prompter.InputSecret("", locale.T("password_prompt_signup"), prompt.InputRequired)
		if err != nil {
			return err
		}
	}

	for i := 0; i < matchTries; i++ {
		confirmMsg := locale.T("password_prompt_confirm")
		input.Password2, err = prompter.InputSecret("", confirmMsg, prompt.InputRequired)
		if err != nil {
			return err
		}

		if input.Password2 == input.Password {
			break
		}

		locErrMsgID := "err_password_confirmation_failed"
		if i < matchTries-1 {
			out.Notice(locale.T(locErrMsgID))
			continue
		}
		return locale.NewError(locErrMsgID)
	}

	input.Email, err = prompter.Input("", locale.T("email_prompt"), new(string), prompt.InputRequired)
	if err != nil {
		return err
	}
	return nil
}

func doSignup(input *signupInput, out output.Outputer, auth *authentication.Auth) error {
	params := users.NewAddUserParams()
	eulaHelper := true
	params.SetUser(&mono_models.UserEditable{
		Email:        input.Email,
		Username:     input.Username,
		Password:     input.Password,
		Name:         input.Username,
		EULAAccepted: &eulaHelper,
	})
	addUserOK, err := mono.Get().Users.AddUser(params)

	// Error checking
	if err != nil {
		switch err.(type) {
		// Authentication failed due to email already existing (username check already happened at this point)
		case *users.AddUserConflict:
			return locale.WrapInputError(err, "err_auth_signup_user_exists", "", api.ErrorMessageFromPayload(err))
		case *users.AddUserBadRequest:
			return locale.WrapInputError(err, "err_auth_signup_bad_request", "", api.ErrorMessageFromPayload(err))
		default:
			multilog.Error("Encountered unknown error adding user: %v", err)
			return locale.WrapError(err, "err_auth_failed_unknown_cause", "", api.ErrorMessageFromPayload(err))
		}
	}

	err = AuthenticateWithCredentials(&mono_models.Credentials{
		Username: input.Username,
		Password: input.Password,
	}, auth)
	if err != nil {
		return err
	}

	if err := auth.CreateToken(); err != nil {
		return locale.WrapError(err, "err_auth_token", "Failed to create token after signup")
	}

	out.Notice(locale.T("signup_success", map[string]string{
		"Email": addUserOK.Payload.User.Email,
	}))

	return nil
}

// UsernameValidator verifies whether the chosen username is usable
func UsernameValidator(val interface{}) error {
	value := val.(string)
	params := users.NewUniqueUsernameParams()
	params.SetUsername(value)
	res, err := mono.Get().Users.UniqueUsername(params)
	if err != nil || *res.Payload.Code != int64(200) {
		return locale.NewError("err_username_taken")
	}
	return nil
}
