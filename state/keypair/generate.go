package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/spf13/cobra"
)

// Prompter is accessible so tests can overwrite it with Mock.  Do not use if you're not writing code for this package
var Prompter prompt.Prompter

func init() {
	Prompter = prompt.New()
}

// ExecuteGenerate processes the `keypair generate` sub-command.
func ExecuteGenerate(_ *cobra.Command, args []string) {
	secretsClient := secretsapi.Get()
	var passphrase string
	var failure *failures.Failure
	var encodedKeypair *keypairs.EncodedKeypair

	if Flags.SkipPassphrase {
		// for the moment, we do not want to record any unencrypted private-keys
		Flags.DryRun = true
	}

	if !Flags.DryRun {
		// ensure user is authenticated before bothering to generate keypair and ask for passphrase
		_, failure = secretsClient.AuthenticatedUserID()
	}

	if failure == nil && !Flags.SkipPassphrase {
		passphrase, failure = Prompter.InputSecret(locale.T("passphrase_prompt"), prompt.InputRequired)
	}

	if failure == nil {
		encodedKeypair, failure = keypairs.GenerateEncodedKeypair(passphrase, Flags.Bits)
	}

	if failure == nil {
		if Flags.DryRun {
			print.Line(encodedKeypair.EncodedPrivateKey)
			print.Line(encodedKeypair.EncodedPublicKey)
		} else {
			failure = keypairs.SaveEncodedKeypair(secretsClient, encodedKeypair)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
	} else {
		print.Line(locale.T("keypair_generate_success"))
	}
}
