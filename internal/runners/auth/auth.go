package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcmanager"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type configurable interface {
	keypairs.Configurable
	GetInt(string) int
}

type Auth struct {
	output.Outputer
	*authentication.Auth
	prompt.Prompter
	cfg    configurable
	svcMgr *svcmanager.Manager
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
	primer.Configurer
	primer.Svcer
}

func NewAuth(prime primeable) *Auth {
	return &Auth{
		prime.Output(),
		prime.Auth(),
		prime.Prompt(),
		prime.Config(),
		prime.SvcManager(),
	}
}

type AuthParams struct {
	Token    string
	Username string
	Password string
	Totp     string
}

// Run runs our command
func (a *Auth) Run(params *AuthParams) error {
	if !a.Authenticated() {
		if err := a.authenticate(params); err != nil {
			return locale.WrapError(err, "err_auth_authenticate", "Could not authenticate.")
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
	if params.Token == "" {
		err := authlet.AuthenticateWithInput(params.Username, params.Password, params.Totp, a.cfg, a.Outputer, a.Prompter, a.svcMgr)
		if err != nil {
			return locale.WrapError(err, "login_err_auth")
		}
	} else {
		err := tokenAuth(params.Token)
		if err != nil {
			return locale.WrapError(err, "login_err_auth_token")
		}
	}
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
