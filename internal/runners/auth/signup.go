package auth

import authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"

type Signup struct{}

func NewSignup() *Signup {
	return &Signup{}
}

func (s *Signup) Run() error {
	return runSignup()
}

func runSignup() error {
	fail := authlet.Signup()
	if fail != nil {
		return fail.ToError()
	}

	return nil
}
