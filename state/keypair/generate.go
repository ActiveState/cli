package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/keys"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/spf13/cobra"
)

// ExecuteGenerate processes the `keypair generate` sub-command.
func ExecuteGenerate(_ *cobra.Command, args []string) {
	secretsClient := secretsapi.DefaultClient
	var passphrase string
	var failure *failures.Failure

	if Flags.SkipPassphrase {
		// for the moment, we do not want to record any unencrypted private-keys
		Flags.DryRun = true
	}

	if !Flags.DryRun {
		// ensure user is authenticated before bothering to generate keypair and ask for passphrase
		_, failure = secretsClient.AuthenticatedUserID()
	}

	if failure == nil && !Flags.SkipPassphrase {
		passphrase, failure = promptForPassphrase()
	}

	if failure == nil {
		failure = generateKeypair(secretsClient, passphrase, Flags.Bits, Flags.DryRun)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	}
}

// generateKeypair implements the behavior to generate a new Secrets key-pair on behalf of the user
// and store that back to the Secrets Service. If dry-run is enabled, a keypair will be generated
// and printed, but not stored anywhere (thus, not used).
func generateKeypair(secretsClient *secretsapi.Client, passphrase string, bits int, dryRun bool) *failures.Failure {
	keypair, failure := keypairs.GenerateRSA(bits)
	if failure != nil {
		return failure
	}

	var encodedPrivateKey string
	if passphrase == "" {
		encodedPrivateKey = keypair.EncodePrivateKey()
	} else {
		encodedPrivateKey, failure = keypair.EncryptAndEncodePrivateKey(passphrase)
		if failure != nil {
			return failure
		}
	}

	encodedPublicKey, failure := keypair.EncodePublicKey()
	if failure != nil {
		return failure
	}

	if !dryRun {
		params := keys.NewSaveKeypairParams().WithKeypair(&secretsModels.KeypairChange{
			EncryptedPrivateKey: &encodedPrivateKey,
			PublicKey:           &encodedPublicKey,
		})

		if _, err := secretsClient.Keys.SaveKeypair(params, secretsClient.Auth); err != nil {
			return secretsapi.FailKeypairSave.New("keypair_err_save")
		}
		print.Line(locale.T("keypair_generate_success"))

		// save the keypair locally to avoid authenticating the keypair every time it's used
		if failure = keypairs.SaveWithDefaults(keypair); failure != nil {
			return failure
		}
	} else {
		print.Line(encodedPrivateKey)
		print.Line(encodedPublicKey)
	}

	return nil
}
