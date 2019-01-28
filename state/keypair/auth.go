package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/spf13/cobra"
)

// ExecuteAuth processes the `keypair auth` command.
func ExecuteAuth(_ *cobra.Command, args []string) {
	secretsClient := secretsapi.DefaultClient
	_, failure := secretsClient.AuthenticatedUserID()

	var passphrase string
	var rawKp *secretsModels.Keypair
	var kp keypairs.Keypair

	if failure == nil {
		rawKp, failure = keypairs.FetchRaw(secretsClient)
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
