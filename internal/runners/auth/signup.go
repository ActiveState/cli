package auth

import authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"

type Signup struct{}

func NewSignup() *Signup {
	return &Signup{}
}

func (s *Signup) Run() error {
	authlet.Signup()
	return nil
}
