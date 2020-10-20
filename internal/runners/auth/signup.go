package auth

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
)

type Signup struct {
	output.Outputer
	prompt.Prompter
}

func NewSignup(prime primeable) *Signup {
	return &Signup{prime.Output(), prime.Prompt()}
}

func (s *Signup) Run() error {
	s.Outputer.Notice(output.Title(locale.Tl("signup_title", "Signing Up With The ActiveState Platform")))

	err := authlet.Signup(s.Outputer, s.Prompter)
	if err != nil {
		return err
	}

	return nil
}
