package export

import (
	"github.com/ActiveState/cli/internal/failures"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type JWT struct{}

func NewJWT() *JWT {
	return &JWT{}
}

// Run processes the `export recipe` command.
func (j *JWT) Run() error {
	logging.Debug("Execute")

	if !authentication.Get().Authenticated() {
		return failures.FailUser.New(locale.T("err_command_requires_auth"))
	}

	print.Line(authentication.Get().BearerToken())
	return nil
}
