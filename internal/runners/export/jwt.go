package export

import (
	"github.com/ActiveState/cli/internal/failures"
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
		return failures.FailUser.New(locale.T("err_command_requires_auth"))
	}

	j.Outputer.Notice(output.Title(locale.Tl("export_jwt_title", "Exporting Credentials")))

	token := authentication.Get().BearerToken()
	j.Outputer.Print(output.NewFormatter(token).WithFormat(output.EditorV0FormatName, []byte(token)))
	return nil
}
