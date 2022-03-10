package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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
		return locale.NewInputError("err_auth_invalid_password_param", "[ACTIONABLLE]--password[/RESET] flag requires [ACTIONABLE]--username[/RESET] flag")
	}

	if p.Totp != "" && (p.Username == "" || p.Password == "") {
		return locale.NewInputError("err_auth_invalid_totp_param", "[ACTIONABLE]--totp[/RESET] flag requires both [ACTIONABLE]--username[/RESET] and [ACTIONABLE]--password[/RESET] flags")
	}

	return nil
}

type SignupParams struct {
	Prompt bool
}

// Run runs our command
func (a *Auth) Run(params *AuthParams) error {
	if !a.Authenticated() {
		if err := params.verify(); err != nil {
			return locale.WrapInputError(err, "err_auth_params", "Invalid authentication params")
		}

		if err := a.authenticate(params); err != nil {
			return locale.WrapError(err, "err_auth_authenticate", "Could not authenticate.")
		}

		if err := a.verifyAuthentication(); err != nil {
			return locale.WrapError(err, "err_auth_verify", "Could not verify authentication")
		}
	}

	data, err := a.userData()
	if err != nil {
		return locale.WrapError(err, "err_auth_userdata", "Could not collect information about your account.")
	}

	a.Outputer.Print(
		output.NewFormatter(data).
			WithFormat(output.PlainFormatName, locale.T("logged_in_as", map[string]string{
				"Name": a.Auth.WhoAmI(),
			})),
	)

	return nil
}

func (a *Auth) authenticate(params *AuthParams) error {
	if params.Prompt || params.Username != "" {
		return authlet.AuthenticateWithInput(params.Username, params.Password, params.Totp, params.NonInteractive, a.Cfg, a.Outputer, a.Prompter, a.Auth)
	}

	if params.Token != "" {
		return authlet.AuthenticateWithToken(params.Token, a.Auth)
	}

	if params.NonInteractive {
		return locale.NewInputError("err_auth_needinput")
	}

	return authlet.AuthenticateWithBrowser(a.Outputer, a.Auth, a.Prompter)
}

func (a *Auth) verifyAuthentication() error {
	if !a.Auth.Authenticated() {
		return locale.NewInputError("login_err_auth")
	}

	a.Outputer.Notice(output.Heading(locale.Tl("authentication_title", "Authentication")))
	a.Outputer.Notice(locale.T("login_success_welcome_back", map[string]string{
		"Name": a.Auth.WhoAmI(),
	}))

	return nil
}

type userData struct {
	Username        string `json:"username,omitempty"`
	URLName         string `json:"urlname,omitempty"`
	Tier            string `json:"tier,omitempty"`
	PrivateProjects bool   `json:"privateProjects"`
}

func (a *Auth) userData() (*userData, error) {
	username := a.Auth.WhoAmI()
	organization, err := model.FetchOrgByURLName(username)
	if err != nil {
		return nil, err
	}

	tiers, err := model.FetchTiers()
	if err != nil {
		return nil, err
	}

	tier := organization.Tier
	privateProjects := false
	for _, t := range tiers {
		if tier == t.Name && t.RequiresPayment {
			privateProjects = true
			break
		}

	}

	return &userData{username, organization.URLname, tier, privateProjects}, nil
}
