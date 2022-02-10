package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
)

type Signup struct {
	output.Outputer
	prompt.Prompter
	keypairs.Configurable
}

func NewSignup(prime primeable) *Signup {
	return &Signup{prime.Output(), prime.Prompt(), prime.Config()}
}

func (s *Signup) Run(interactive bool) error {
	return authlet.Signup(s.Configurable, s.Outputer, s.Prompter, interactive)
}
