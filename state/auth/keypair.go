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
		processExistingKeypairForUser(keypairRes, passphrase)
	} else if secretsapi.FailKeypairNotFound.Matches(failure.Type) {
		generateKeypairForUser(passphrase)
	} else {
		failures.Handle(failure, locale.T("keypair_err"))
	}

	if failures.Handled() != nil {
		doLogout()
		print.Line(locale.T("auth_unresolved_keypair_issue_message"))
	}
}

// generateKeypairForUser attempts to generate and save a Keypair for the currently authenticated user.
func generateKeypairForUser(passphrase string) {
	_, failure := keypairs.GenerateAndSaveEncodedKeypair(secretsapi.DefaultClient, passphrase, constants.DefaultRSABitLength)
	if failure != nil {
		failures.Handle(failure, locale.T("keypair_err_save"))
	} else {
		print.Line(locale.T("keypair_generate_success"))
	}
}

func validateLocalPrivateKey(publicKey string) bool {
	localKeypair, failure := keypairs.LoadWithDefaults()
	return failure == nil && localKeypair.MatchPublicKey(publicKey)
}

func processExistingKeypairForUser(keypairRes *secretsModels.Keypair, passphrase string) {
	keypair, failure := keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, passphrase)
	if failure != nil {
		if keypairs.FailKeypairPassphrase.Matches(failure.Type) {
			// failed to decrypt stored private-key with provided passphrase
			var localKeypair keypairs.Keypair
			if localKeypair, failure = keypairs.LoadWithDefaults(); failure == nil && localKeypair.MatchPublicKey(*keypairRes.PublicKey) {
				// locally stored private-key has a matching public-key, encrypt that with new passphrase and upload
				var encodedKeypair *keypairs.EncodedKeypair
				if encodedKeypair, failure = keypairs.EncodeKeypair(localKeypair, passphrase); failure == nil {
					failure = keypairs.SaveEncodedKeypair(secretsapi.DefaultClient, encodedKeypair)
				}
			} else {
				failure = recoverKeypairFromPreviousPassphrase(keypairRes, passphrase)
				if failure != nil && keypairs.FailKeypairPassphrase.Matches(failure.Type) {
					failure = promptUserToRegenerateKeypair(passphrase)
				}
			}
		}

		if failure != nil {
			failures.Handle(failure, locale.T("keypair_err"))
		}
	} else {
		// update the locally stored private-key
		keypairs.SaveWithDefaults(keypair)
	}
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
