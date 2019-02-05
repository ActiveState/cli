package auth

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/surveyor"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// ensureUserKeypair checks to see if the currently authenticated user has a Keypair. If not, one is generated
// and saved.
func ensureUserKeypair(passphrase string) {
	keypairRes, failure := keypairs.FetchRaw(secretsapi.DefaultClient)
	if failure == nil {
		failure = processExistingKeypairForUser(keypairRes, passphrase)
	} else if secretsapi.FailKeypairNotFound.Matches(failure.Type) {
		failure = generateKeypairForUser(passphrase)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err"))
		doLogout()
		print.Line(locale.T("auth_unresolved_keypair_issue_message"))
	}
}

// generateKeypairForUser attempts to generate and save a Keypair for the currently authenticated user.
func generateKeypairForUser(passphrase string) *failures.Failure {
	_, failure := keypairs.GenerateAndSaveEncodedKeypair(secretsapi.DefaultClient, passphrase, constants.DefaultRSABitLength)
	if failure != nil {
		return failure
	}
	print.Line(locale.T("keypair_generate_success"))
	return nil
}

func validateLocalPrivateKey(publicKey string) bool {
	localKeypair, failure := keypairs.LoadWithDefaults()
	return failure == nil && localKeypair.MatchPublicKey(publicKey)
}

// processExistingKeypairForUser will attempt to ensure the stored private-key for the user is encrypted
// using the provided passphrase. If passphrase match fails, processExistingKeypairForUser will then try
// validate that the locally stored private-key has a public-key matching the one provided in the keypair.
// If public-keys match, the locally stored private-key will be encrypted with the provided passphrase
// and uploaded for the user.
//
// If the previous paths result in failure, user is prompted for their previous passphrase in attempt to
// determine if the password has changed. If successful, private-key is encrypted with passphrase provided
// to this function and uploaded.
//
// If all paths fail, user is prompted to regenerate their keypair which will be encrypted with the
// provided passphrase and then uploaded; unless the user declines, which results in failure.
func processExistingKeypairForUser(keypairRes *secretsModels.Keypair, passphrase string) *failures.Failure {
	keypair, failure := keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, passphrase)
	if failure == nil {
		// yay, store keypair locally just in case it isn't
		return keypairs.SaveWithDefaults(keypair)
	} else if !keypairs.FailKeypairPassphrase.Matches(failure.Type) {
		// failure did not involve an unmatched passphrase
		return failure
	}

	// failed to decrypt stored private-key with provided passphrase, try using a local private-key
	var localKeypair keypairs.Keypair
	localKeypair, failure = keypairs.LoadWithDefaults()
	if failure == nil && localKeypair.MatchPublicKey(*keypairRes.PublicKey) {
		// locally stored private-key has a matching public-key, encrypt that with new passphrase and upload
		var encodedKeypair *keypairs.EncodedKeypair
		if encodedKeypair, failure = keypairs.EncodeKeypair(localKeypair, passphrase); failure != nil {
			return failure
		}
		return keypairs.SaveEncodedKeypair(secretsapi.DefaultClient, encodedKeypair)
	}

	// failed to validate with local private-key, try using previous passphrase
	failure = recoverKeypairFromPreviousPassphrase(keypairRes, passphrase)
	if failure != nil && keypairs.FailKeypairPassphrase.Matches(failure.Type) {
		// that failed, see if they want to regenerate their passphrase
		failure = promptUserToRegenerateKeypair(passphrase)
	}
	return failure
}

func recoverKeypairFromPreviousPassphrase(keypairRes *secretsModels.Keypair, passphrase string) *failures.Failure {
	print.Line(locale.T("previous_password_message"))
	prevPassphrase, failure := promptForPreviousPassphrase()
	if failure == nil {
		var keypair keypairs.Keypair
		keypair, failure = keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, prevPassphrase)
		if failure == nil {
			// previous passphrase is valid, encrypt private-key with new passphrase and upload
			encodedKeypair, failure := keypairs.EncodeKeypair(keypair, passphrase)
			if failure == nil {
				failure = keypairs.SaveEncodedKeypair(secretsapi.DefaultClient, encodedKeypair)
			}
		}
	}
	return failure
}

func promptForPreviousPassphrase() (string, *failures.Failure) {
	var passphrase string
	var prompt = &survey.Password{Message: locale.T("previous_password_prompt")}
	if err := survey.AskOne(prompt, &passphrase, surveyor.ValidateRequired); err != nil {
		return "", failures.FailUserInput.New("auth_err_password_prompt")
	}
	return passphrase, nil
}

func promptUserToRegenerateKeypair(passphrase string) *failures.Failure {
	var failure *failures.Failure
	// previous passphrase is invalid, inform user and ask if they want to generate a new keypair
	print.Line(locale.T("auth_generate_new_keypair_message"))
	if surveyor.Confirm("auth_confirm_generate_new_keypair_prompt") {
		_, failure = keypairs.GenerateAndSaveEncodedKeypair(secretsapi.DefaultClient, passphrase, constants.DefaultRSABitLength)
		// TODO delete user's secrets
	} else {
		failure = keypairs.FailKeypair.New("auth_err_unrecoverable_keypair")
	}
	return failure
}
