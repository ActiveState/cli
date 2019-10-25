package auth

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func tokenAuth() {
	auth := authentication.Get()
	fail := auth.AuthenticateWithModel(&mono_models.Credentials{
		Token: Flags.Token,
	})

	if fail != nil {
		failures.Handle(fail.ToError(), locale.T("err_auth_failed_unknown_cause"))
		return
	}
}
