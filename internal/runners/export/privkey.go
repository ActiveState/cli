package export

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type PrivateKey struct{}

func NewPrivateKey() *PrivateKey {
	return &PrivateKey{}
}

type PrivateKeyParams struct {
	Auth *authentication.Auth
}

// Run processes the `export recipe` command.
func (p *PrivateKey) Run(params *PrivateKeyParams) error {
	logging.Debug("Execute")

	if !params.Auth.Authenticated() {
		return failures.FailUser.New(locale.T("err_command_requires_auth"))
	}

	filepath := keypairs.LocalKeyFilename(constants.KeypairLocalFileName)
	contents, fail := fileutils.ReadFile(filepath)
	if fail != nil {
		return fail
	}

	print.Line(string(contents))
	return nil
}
