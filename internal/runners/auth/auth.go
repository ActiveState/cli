package auth

import (
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Auth struct{}

func NewAuth() *Auth {
	return &Auth{}
}

type AuthParams struct {
	Output   string
	Token    string
	Username string
	Password string
	Totp     string
}

// Run runs our command
func (a *Auth) Run(params *AuthParams) error {
	return runAuth(params)
}

func runAuth(params *AuthParams) error {
	auth := authentication.Get()

	output := commands.Output(strings.ToLower(params.Output))
	if !auth.Authenticated() {
		return authenticate(params, auth)
	}

	logging.Debug("Already authenticated")
	switch output {
	case commands.JSON, commands.EditorV0:
		user, fail := userToJSON(auth.WhoAmI())
		if fail != nil {
			return fail.WithDescription("login_err_output")
		}
		print.Line(string(user))
	default:
		print.Line(locale.T("logged_in_as", map[string]string{
			"Name": auth.WhoAmI(),
		}))
	}

	return nil
}

func authenticate(params *AuthParams, auth *authentication.Auth) error {
	if params.Token == "" {
		fail := authlet.AuthenticateWithInput(params.Username, params.Password, params.Totp)
		if fail != nil {
			return fail.WithDescription("login_err_auth")
		}
	} else {
		fail := tokenAuth(params.Token)
		if fail != nil {
			return fail.WithDescription("login_err_auth_token")
		}
	}
	if !auth.Authenticated() {
		return failures.FailUser.New(locale.T("login_err_auth"))
	}

	switch commands.Output(strings.ToLower(params.Output)) {
	case commands.JSON, commands.EditorV0:
		user, fail := userToJSON(auth.WhoAmI())
		if fail != nil {
			return fail.WithDescription("login_err_output")
		}
		print.Line(string(user))
	default:
		print.Line(locale.T("login_success_welcome_back", map[string]string{
			"Name": auth.WhoAmI(),
		}))
	}

	return nil
}

func userToJSON(username string) ([]byte, *failures.Failure) {
	type userJSON struct {
		Username        string `json:"username,omitempty"`
		URLName         string `json:"urlname,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}

	organization, fail := model.FetchOrgByURLName(username)
	if fail != nil {
		return nil, fail
	}

	tiers, fail := model.FetchTiers()
	if fail != nil {
		return nil, fail
	}

	tier := organization.Tier
	privateProjects := false
	for _, t := range tiers {
		privateProjects = (tier == t.Name && t.RequiresPayment)
	}

	userJ := userJSON{username, organization.URLname, tier, privateProjects}
	bs, err := json.Marshal(userJ)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return bs, nil
}
