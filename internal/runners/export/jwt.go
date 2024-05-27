package export

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type JWT struct {
	output.Outputer
	*authentication.Auth
}

func NewJWT(prime primeable) *JWT {
	return &JWT{prime.Output(), prime.Auth()}
}

type JWTParams struct {
}

// Run processes the `export recipe` command.
func (j *JWT) Run(params *JWTParams) error {
	logging.Debug("Execute")

	if !j.Auth.Authenticated() {
		return locale.NewInputError("err_jwt_not_authenticated")
	}

	token := j.Auth.BearerToken()
	j.Outputer.Print(output.Prepare(
		token,
		&struct {
			Value string `json:"value"`
		}{token},
	))
	return nil
}
