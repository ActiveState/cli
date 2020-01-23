package auth

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func totpAuth(params *AuthParams) *failures.Failure {
	if params.Username == "" || params.Password == "" {
		return failures.FailUser.New(locale.T("login_err_auth_totp_params"))
	}

	auth := authentication.Get()
	return auth.AuthenticateWithModel(&mono_models.Credentials{
		Username: params.Username,
		Password: params.Password,
		Totp:     params.Totp,
	})
}
