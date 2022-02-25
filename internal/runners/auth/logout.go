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
	Cfg keypairs.Configurable
}

func NewLogout(prime primeable) *Logout {
	return &Logout{prime.Output(), prime.Auth(), prime.Config()}
}

func (l *Logout) Run() error {
	l.Auth.Logout()
	err := keypairs.DeleteWithDefaults(l.Cfg)
	if err != nil {
		return locale.WrapError(err, "err_auth_logout", "Failed to delete authentication key")
	}
	l.Outputer.Notice(output.Heading(locale.Tl("authentication_title", "Authentication")))
	if l.Auth.AvailableAPIToken() == "" {
		l.Outputer.Notice(locale.T("logged_out"))
	} else {
		l.Outputer.Notice(locale.T("logout_still_have_api_token"))
	}
	return nil
}
