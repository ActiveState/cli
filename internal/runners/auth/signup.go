package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Signup struct {
	output.Outputer
	prompt.Prompter
	keypairs.Configurable
	*authentication.Auth
}

type SignupParams struct {
	Prompt bool
}

func NewSignup(prime primeable) *Signup {
	return &Signup{prime.Output(), prime.Prompt(), prime.Config(), prime.Auth()}
}

func (s *Signup) Run(params *SignupParams) error {
	if s.Auth.Authenticated() {
		return locale.NewInputError("err_auth_authenticated", "You are already authenticated as: {{.V0}}. You can log out by running '[ACTIONABLE]state auth logout[/RESET]'.", s.Auth.WhoAmI())
	}

	return auth.SignupWithBrowser(s.Outputer, s.Auth, s.Prompter, s.Configurable)
}
