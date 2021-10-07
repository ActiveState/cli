package auth

import (
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcmanager"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
)

type Signup struct {
	output.Outputer
	prompt.Prompter
	cnf    configurable
	svcMgr *svcmanager.Manager
}

func NewSignup(prime primeable) *Signup {
	return &Signup{
		prime.Output(),
		prime.Prompt(),
		prime.Config(),
		prime.SvcManager(),
	}
}

func (s *Signup) Run() error {
	err := authlet.Signup(s.cnf, s.Outputer, s.Prompter, s.svcMgr)
	if err != nil {
		return err
	}

	return nil
}
