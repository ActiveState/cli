package auth

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
)

func tokenAuth() {
	doTokenAuth()
}

func doTokenAuth() {
	loginOK, err := api.Authenticate(&models.Credentials{
		Token: Args.Token,
	})

	// Error checking
	if err != nil {
		switch err.(type) {
		default:
			failures.Handle(err, locale.T("err_auth_failed_unknown_cause"))
			return
		}
	}

	print.Info(locale.T("login_success_welcome_back", map[string]string{
		"Name": loginOK.Payload.User.Username,
	}))
}
