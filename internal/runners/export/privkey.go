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
	cfg keypairs.Configurable
	*authentication.Auth
}

func NewPrivateKey(prime primeable) *PrivateKey {
	return &PrivateKey{prime.Output(), prime.Config(), prime.Auth()}
}

type PrivateKeyParams struct {
}

// Run processes the `export recipe` command.
func (p *PrivateKey) Run(params *PrivateKeyParams) error {
	logging.Debug("Execute")

	if !p.Auth.Authenticated() {
		return locale.NewInputError("err_export_privkey_requires_auth",
			"You need to be authenticated to run this command. Authenticate by running '[ACTIONABLE]state auth --prompt[/RESET]'.")
	}

	filepath := keypairs.LocalKeyFilename(p.cfg.ConfigPath(), constants.KeypairLocalFileName)
	if !fileutils.FileExists(filepath) {
		return locale.NewInputError("err_privkey_nofile",
			"No private key file exists. Please make sure you have authenticated with your password. Authenticating with an API key is not sufficient.")
	}

	contents, err := fileutils.ReadFile(filepath)
	if err != nil {
		return err
	}

	p.Outputer.Print(output.Prepare(
		string(contents),
		&struct {
			Value string `json:"value"`
		}{string(contents)},
	))
	return nil
}
