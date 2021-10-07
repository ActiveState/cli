package auth

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcmanager"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Logout struct {
	output.Outputer
	*authentication.Auth
	cfg    configurable
	svcMgr *svcmanager.Manager
}

func NewLogout(prime primeable) *Logout {
	return &Logout{
		prime.Output(),
		prime.Auth(),
		prime.Config(),
		prime.SvcManager(),
	}
}

func (l *Logout) Run() error {
	if err := authlet.Logout(l.cfg, l.svcMgr); err != nil {
		return locale.WrapError(err, "err_auth_logout", "Failed to delete authentication key")
	}
	l.Outputer.Notice(output.Heading(locale.Tl("authentication_title", "Authentication")))
	l.Outputer.Notice(locale.T("logged_out"))
	return nil
}
