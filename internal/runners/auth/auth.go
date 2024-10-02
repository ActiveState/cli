package auth

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Auth struct {
	output.Outputer
	*authentication.Auth
	prompt.Prompter
	Cfg keypairs.Configurable
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
	primer.Configurer
}

func NewAuth(prime primeable) *Auth {
	return &Auth{prime.Output(), prime.Auth(), prime.Prompt(), prime.Config()}
}

type AuthParams struct {
	Token          string
	Username       string
	Password       string
	Totp           string
	Prompt         bool
	NonInteractive bool
}

func (p AuthParams) verify() error {
	if p.Username != "" && p.Password == "" {
		return locale.NewInputError("err_auth_invalid_username_param", "[ACTIONABLE]--username[/RESET] flag requires [ACTIONABLE]--password[/RESET] flag")
	}

	if p.Username == "" && p.Password != "" {
		return locale.NewInputError("err_auth_invalid_password_param", "[ACTIONABLE]--password[/RESET] flag requires [ACTIONABLE]--username[/RESET] flag")
	}

	if p.Totp != "" && (p.Username == "" || p.Password == "") {
		return locale.NewInputError("err_auth_invalid_totp_param", "[ACTIONABLE]--totp[/RESET] flag requires both [ACTIONABLE]--username[/RESET] and [ACTIONABLE]--password[/RESET] flags")
	}

	return nil
}

// Run runs our command
func (a *Auth) Run(params *AuthParams) error {
	if !a.Authenticated() {
		if err := params.verify(); err != nil {
			return locale.WrapError(err, "err_auth_params", "Invalid authentication params")
		}

		if err := a.authenticate(params); err != nil {
			return locale.WrapError(err, "err_auth_authenticate", "Could not authenticate.")
		}

		if err := a.verifyAuthentication(); err != nil {
			return locale.WrapError(err, "err_auth_verify", "Could not verify authentication")
		}
	}

	username := a.Auth.WhoAmI()
	a.Outputer.Print(output.Prepare(
		locale.T("logged_in_as", map[string]string{"Name": username}),
		&struct {
			Username string `json:"username"`
		}{
			username,
		},
	))

	return nil
}

func (a *Auth) authenticate(params *AuthParams) error {
	if params.Prompt || params.Username != "" {
		return auth.AuthenticateWithInput(params.Username, params.Password, params.Totp, params.NonInteractive, a.Cfg, a.Outputer, a.Prompter, a.Auth)
	}

	if params.Token != "" {
		return auth.AuthenticateWithToken(params.Token, a.Auth)
	}

	if apiKey := os.Getenv(constants.APIKeyEnvVarName); apiKey != "" {
		err := auth.AuthenticateWithToken(apiKey, a.Auth)
		if err != nil {
			return locale.WrapError(err, "err_auth_api_key", "Failed to authenticate with [ACTIONABLE]{{.V0}}[/RESET] environment variable", constants.APIKeyEnvVarName)
		}
		return nil
	}

	if params.NonInteractive {
		return locale.NewInputError("err_auth_needinput")
	}

	return auth.AuthenticateWithBrowser(a.Outputer, a.Auth, a.Prompter, a.Cfg)
}

func (a *Auth) verifyAuthentication() error {
	if !a.Auth.Authenticated() {
		return locale.NewInputError("login_err_auth")
	}

	a.Outputer.Notice(output.Title(locale.Tl("authentication_title", "Authentication")))
	a.Outputer.Notice(locale.T("login_success_welcome_back", map[string]string{
		"Name": a.Auth.WhoAmI(),
	}))

	return nil
}
