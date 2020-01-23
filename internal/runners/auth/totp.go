package auth

import (
	"github.com/ActiveState/cli/internal/failures"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func totpAuth(params *AuthParams) *failures.Failure {
	auth := authentication.Get()
	return auth.AuthenticateWithModel(&mono_models.Credentials{
		Username: params.Username,
		Password: params.Password,
		Totp:     params.Totp,
	})
}
