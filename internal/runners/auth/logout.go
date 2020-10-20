package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Logout struct {
	output.Outputer
	*authentication.Auth
}

func NewLogout(prime primeable) *Logout {
	return &Logout{prime.Output(), prime.Auth()}
}

func (l *Logout) Run() error {
	l.Outputer.Notice(output.Title(locale.Tl("logout_title", "Logging Out Of The ActiveState Platform")))

	l.Auth.Logout()
	keypairs.DeleteWithDefaults()
	l.Outputer.Notice(output.Heading(locale.Tl("authentication_title", "Authentication")))
	l.Outputer.Notice(locale.T("logged_out"))
	return nil
}
