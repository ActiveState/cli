package export

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type PrivateKey struct {
	output.Outputer
	*authentication.Auth
}

func NewPrivateKey(prime primeable) *PrivateKey {
	return &PrivateKey{prime.Output(), prime.Auth()}
}

type PrivateKeyParams struct {
}

// Run processes the `export recipe` command.
func (p *PrivateKey) Run(params *PrivateKeyParams) error {
	logging.Debug("Execute")

	if !p.Auth.Authenticated() {
		return locale.NewError("User")
	}

	filepath := keypairs.LocalKeyFilename(constants.KeypairLocalFileName)
	contents, err := fileutils.ReadFile(filepath)
	if err != nil {
		return err
	}

	p.Outputer.Print(string(contents))
	return nil
}
