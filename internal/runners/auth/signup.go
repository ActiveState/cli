package auth

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Signup struct {
	output.Outputer
	prompt.Prompter
	cfg *config.Instance
}

func NewSignup(prime primeable) *Signup {
	return &Signup{prime.Output(), prime.Prompt(), prime.Config()}
}

func (s *Signup) Run(params *SignupParams) error {
	auth := authentication.New(s.cfg)
	defer auth.Close()
	if auth.Authenticated() {
		return locale.NewInputError("err_auth_authenticated", "You are already authenticated as: {{.V0}}. You can log out by running `state auth logout`.", auth.WhoAmI())
	}

	if !params.Interactive {
		return authlet.AuthenticateWithDevice(s.Outputer) // user can sign up from this page too
	}
	return authlet.Signup(s.cfg, s.Outputer, s.Prompter)
}
