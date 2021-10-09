package auth

import (
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func tokenAuth(rep Reporter, token string) error {
	auth := authentication.LegacyGet(rep)
	return auth.AuthenticateWithModel(&mono_models.Credentials{
		Token: token,
	})
}
