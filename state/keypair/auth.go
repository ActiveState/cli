package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/spf13/cobra"
)

// ExecuteAuth processes the `keypair auth` command.
func (cmd *Command) ExecuteAuth(_ *cobra.Command, args []string) {
	_, failure := cmd.secretsClient.Authenticated()

	var passphrase string
	var rawKp *secretsModels.Keypair
	var kp keypairs.Keypair

	if failure == nil {
		rawKp, failure = keypairs.FetchRaw(cmd.secretsClient)
	}

	if failure == nil {
		passphrase, failure = promptForPassphrase()
	}

	if failure == nil {
		kp, failure = keypairs.ParseEncryptedRSA(*rawKp.EncryptedPrivateKey, passphrase)
	}

	if failure == nil {
		keypairs.SaveWithDefaults(kp)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}
