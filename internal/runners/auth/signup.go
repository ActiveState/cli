package auth

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/svcmanager"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
)

type Signup struct {
	output.Outputer
	prompt.Prompter
	keypairs.Configurable
	cnf    *config.Instance
	svcMgr *svcmanager.Manager
}

func NewSignup(prime primeable) *Signup {
	cnf := prime.Config()
	return &Signup{
		prime.Output(),
		prime.Prompt(),
		cnf,
		cnf,
		prime.SvcManager(),
	}
}

func (s *Signup) Run() error {
	err := authlet.Signup(s.Configurable, s.Outputer, s.Prompter, s.cnf, s.svcMgr)
	if err != nil {
		return err
	}

	return nil
}
