package auth

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

var (
	// FailInvalidPassword indicates the users desired password is invalid
	FailInvalidPassword = failures.Type("auth.failure.invalidpassword")

	// FailAddUserConflict indicates a failure due to an existing user
	FailAddUserConflict = failures.Type("auth.failure.adduserconflict")
)

type signupInput struct {
	Name      string
	Email     string
	Username  string
	Password  string
	Password2 string
}

// Signup will prompt the user to create an account
func Signup(out output.Outputer, prompt prompt.Prompter) error {
	input := &signupInput{}

	accepted, fail := promptTOS(out, prompt)
	if fail != nil {
		return fail
	}
	if !accepted {
		return locale.NewInputError("tos_not_accepted", "")
	}

	fail = promptForSignup(input, out, prompt)
	if fail != nil {
		return fail.WithDescription("err_prompt_unknown").ToError()
	}

	if fail = doSignup(input, out); fail != nil {
		return fail.ToError()
	}

	if authentication.Get().Authenticated() {
		if fail := generateKeypairForUser(input.Password); fail != nil {
			return fail.WithDescription("keypair_err_save").ToError()
		}
	}

	return nil
}

func signupFromLogin(username string, password string, out output.Outputer, prompt prompt.Prompter) *failures.Failure {
	input := &signupInput{}

	input.Username = username
	input.Password = password
	err := promptForSignup(input, out, prompt)
	if err != nil {
		return failures.FailUserInput.Wrap(err)
	}

	return doSignup(input, out)
}

func downloadTOS() (string, *failures.Failure) {
	resp, err := http.Get(constants.TermsOfServiceURLText)
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}
	defer resp.Body.Close()

	tosPath := filepath.Join(config.ConfigPath(), "platform_tos.txt")
	tosFile, err := os.Create(tosPath)
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}
	defer tosFile.Close()

	_, err = io.Copy(tosFile, resp.Body)
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}

	return tosPath, nil
}

func promptTOS(out output.Outputer, prompt prompt.Prompter) (bool, *failures.Failure) {
	choices := []string{
		locale.T("tos_accept"),
		locale.T("tos_not_accept"),
		locale.T("tos_show_full"),
	}

	out.Notice(locale.Tl("tos_disclaimer", constants.TermsOfServiceURLLatest))
	choice, fail := prompt.Select(locale.T("tos_acceptance"), choices, locale.T("tos_accept"))
	if fail != nil {
		return false, fail
	}

	switch choice {
	case locale.T("tos_accept"):
		return true, nil
	case locale.T("tos_not_accept"):
		return false, nil
	case locale.T("tos_show_full"):
		tosFilePath, fail := downloadTOS()
		if fail != nil {
			return false, fail.WithDescription("err_download_tos")
		}

		tos, err := ioutil.ReadFile(tosFilePath)
		if err != nil {
			return false, failures.FailIO.Wrap(err)
		}
		out.Print(tos)
		return prompt.Confirm(locale.T("tos_acceptance"), true)
	}

	return false, nil
}

func promptForSignup(input *signupInput, out output.Outputer, prompter prompt.Prompter) *failures.Failure {
	var fail *failures.Failure

	if input.Username != "" {
		out.Notice(locale.T("confirm_password_account_creation"))
	} else {
		input.Username, fail = prompter.Input(locale.T("username_prompt_signup"), "", prompt.InputRequired)
		if fail != nil {
			return fail
		}
		input.Password, fail = prompter.InputSecret(locale.T("password_prompt_signup"), prompt.InputRequired)
		if fail != nil {
			return fail
		}
	}

	// Must define password validator here as it has to reference the input
	var passwordValidator = func(val interface{}) error {
		value := val.(string)
		if value != input.Password {
			return FailInvalidPassword.New(locale.T("err_password_confirmation_failed"))
		}
		return nil
	}

	input.Password2, fail = prompter.InputSecret(locale.T("password_prompt_confirm"), prompt.InputRequired)
	if fail != nil {
		return fail
	}
	err := passwordValidator(input.Password2)
	if err != nil {
		return FailInvalidPassword.Wrap(err)
	}

	input.Name, fail = prompter.Input(locale.T("name_prompt"), "", prompt.InputRequired)
	if fail != nil {
		return fail
	}

	input.Email, fail = prompter.Input(locale.T("email_prompt"), "", prompt.InputRequired)
	if fail != nil {
		return fail
	}
	return nil
}

func doSignup(input *signupInput, out output.Outputer) *failures.Failure {
	params := users.NewAddUserParams()
	eulaHelper := true
	params.SetUser(&mono_models.UserEditable{
		Email:        input.Email,
		Username:     input.Username,
		Password:     input.Password,
		Name:         input.Name,
		EULAAccepted: &eulaHelper,
	})
	addUserOK, err := mono.Get().Users.AddUser(params)

	// Error checking
	if err != nil {
		switch err.(type) {
		// Authentication failed due to email already existing (username check already happened at this point)
		case *users.AddUserConflict:
			logging.Error("Encountered add user conflict: %v", err)
			return FailAddUserConflict.New(locale.T("err_auth_signup_user_exists", api.ErrorMessageFromPayload(err)))
		default:
			logging.Error("Encountered unknown error adding user: %v", err)
			return FailAuthUnknown.New(locale.T("err_auth_failed_unknown_cause", api.ErrorMessageFromPayload(err)))
		}
	}

	fail := AuthenticateWithCredentials(&mono_models.Credentials{
		Username: input.Username,
		Password: input.Password,
	})
	if fail != nil {
		return fail
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
		return errors.New(locale.T("err_username_taken"))
	}
	return nil
}
