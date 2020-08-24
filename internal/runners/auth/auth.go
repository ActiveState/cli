package auth

import (
	"github.com/ActiveState/cli/internal/failures"
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
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
}

func NewAuth(prime primeable) *Auth {
	return &Auth{prime.Output(), prime.Auth(), prime.Prompt()}
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
			return locale.WrapError(err, locale.Tl("err_auth_authenticate", "Could not authenticate."))
		}
	}

	data, err := a.userData()
	if err != nil {
		return locale.WrapError(err, locale.Tl("err_auth_userdata", "Could not collect information about your account."))
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
		fail := authlet.AuthenticateWithInput(params.Username, params.Password, params.Totp, a.Outputer, a.Prompter)
		if fail != nil {
			return fail.WithDescription("login_err_auth").ToError()
		}
	} else {
		fail := tokenAuth(params.Token)
		if fail != nil {
			return fail.WithDescription("login_err_auth_token").ToError()
		}
	}
	if !a.Auth.Authenticated() {
		return failures.FailUser.New(locale.T("login_err_auth")).ToError()
	}
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
	organization, fail := model.FetchOrgByURLName(username)
	if fail != nil {
		return nil, fail.ToError()
	}

	tiers, fail := model.FetchTiers()
	if fail != nil {
		return nil, fail.ToError()
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
