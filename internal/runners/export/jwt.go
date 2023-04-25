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

type jwtOutput struct {
	Value string `json:"value"`
}

func (f *jwtOutput) MarshalOutput(format output.Format) interface{} {
	return f.Value
}

func (f *jwtOutput) MarshalStructured(format output.Format) interface{} {
	if format == output.EditorV0FormatName {
		return []byte(f.Value)
	}
	return f
}

// Run processes the `export recipe` command.
func (j *JWT) Run(params *JWTParams) error {
	logging.Debug("Execute")

	if !j.Auth.Authenticated() {
		return locale.NewInputError("err_jwt_not_authenticated")
	}

	token := authentication.LegacyGet().BearerToken()
	j.Outputer.Print(&jwtOutput{token})
	return nil
}
