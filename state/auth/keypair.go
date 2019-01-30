package auth

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
)

// ensureUserKeypair checks to see if the currently authenticated user has a Keypair. If not, one is generated
// and saved.
func ensureUserKeypair(passphrase string) {
	_, failure := keypairs.FetchRaw(secretsapi.DefaultClient)
	if failure != nil {
		if secretsapi.FailKeypairNotFound.Matches(failure.Type) {
			generateKeypairForUser(passphrase)
		} else {
			failures.Handle(failure, locale.T("keypair_err"))
		}
	}
}

// generateKeypairForUser attempts to generate and save a Keypair for the currently authenticated user.
func generateKeypairForUser(passphrase string) {
	_, failure := keypairs.GenerateAndSaveEncodedKeypair(secretsapi.DefaultClient, passphrase, constants.DefaultRSABitLength)
	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err_save"))
	}
}
