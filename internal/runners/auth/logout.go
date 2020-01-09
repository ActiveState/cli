package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Logout struct{}

func NewLogout() *Logout {
	return &Logout{}
}

func (l *Logout) Run() error {
	return runLogout()
}

func runLogout() error {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
	print.Line(locale.T("logged_out"))
	return nil
}
