package auth

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
)

// ensureUserKeypair checks to see if the currently authenticated user has a Keypair. If not, one is generated
// and saved.
func ensureUserKeypair(passphrase string, out output.Outputer, prompt prompt.Prompter) error {
	keypairRes, err := keypairs.FetchRaw(secretsapi.Get())
	if err == nil {
		err = processExistingKeypairForUser(keypairRes, passphrase, out, prompt)
	} else if errs.Matches(err, &keypairs.ErrKeypairNotFound{}) {
		err = generateKeypairForUser(passphrase)
	}

	if err != nil {
		Logout()
		out.Error(locale.T("auth_unresolved_keypair_issue_message"))
		return err
	}

	return nil
}

// generateKeypairForUser attempts to generate and save a Keypair for the currently authenticated user.
func generateKeypairForUser(passphrase string) error {
	_, err := keypairs.GenerateAndSaveEncodedKeypair(secretsapi.Get(), passphrase, constants.DefaultRSABitLength)
	if err != nil {
		return err
	}
	return nil
}

func validateLocalPrivateKey(publicKey string) bool {
	localKeypair, err := keypairs.LoadWithDefaults()
	return err == nil && localKeypair.MatchPublicKey(publicKey)
}

// processExistingKeypairForUser will attempt to ensure the stored private-key for the user is encrypted
// using the provided passphrase. If passphrase match fails, processExistingKeypairForUser will then try
// validate that the locally stored private-key has a public-key matching the one provided in the keypair.
// If public-keys match, the locally stored private-key will be encrypted with the provided passphrase
// and uploaded for the user.
//
// If the previous paths result in err, user is prompted for their previous passphrase in attempt to
// determine if the password has changed. If successful, private-key is encrypted with passphrase provided
// to this function and uploaded.
//
// If all paths err, user is prompted to regenerate their keypair which will be encrypted with the
// provided passphrase and then uploaded; unless the user declines, which results in err.
func processExistingKeypairForUser(keypairRes *secretsModels.Keypair, passphrase string, out output.Outputer, prompt prompt.Prompter) error {
	keypair, err := keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, passphrase)
	if err == nil {
		// yay, store keypair locally just in case it isn't
		return keypairs.SaveWithDefaults(keypair)
	} else if !errs.Matches(err, &keypairs.ErrKeypairPassphrase{}) {
		// err did not involve an unmatched passphrase
		return err
	}

	// failed to decrypt stored private-key with provided passphrase, try using a local private-key
	var localKeypair keypairs.Keypair
	localKeypair, err = keypairs.LoadWithDefaults()
	if err == nil && localKeypair.MatchPublicKey(*keypairRes.PublicKey) {
		// locally stored private-key has a matching public-key, encrypt that with new passphrase and upload
		var encodedKeypair *keypairs.EncodedKeypair
		if encodedKeypair, err = keypairs.EncodeKeypair(localKeypair, passphrase); err != nil {
			return err
		}
		return keypairs.SaveEncodedKeypair(secretsapi.Get(), encodedKeypair)
	}

	// failed to validate with local private-key, try using previous passphrase
	err = recoverKeypairFromPreviousPassphrase(keypairRes, passphrase, out, prompt)
	if err != nil && errs.Matches(err, &keypairs.ErrKeypairPassphrase{}) {
		// that failed, see if they want to regenerate their passphrase
		err = promptUserToRegenerateKeypair(passphrase, out, prompt)
	}
	return err
}

func recoverKeypairFromPreviousPassphrase(keypairRes *secretsModels.Keypair, passphrase string, out output.Outputer, prompt prompt.Prompter) error {
	out.Notice(locale.T("previous_password_message"))
	prevPassphrase, err := promptForPreviousPassphrase(prompt)
	if err == nil {
		var keypair keypairs.Keypair
		keypair, err = keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, prevPassphrase)
		if err == nil {
			// previous passphrase is valid, encrypt private-key with new passphrase and upload
			encodedKeypair, err := keypairs.EncodeKeypair(keypair, passphrase)
			if err == nil {
				err = keypairs.SaveEncodedKeypair(secretsapi.Get(), encodedKeypair)
			}
		}
	}
	return err
}

func promptForPreviousPassphrase(prompt prompt.Prompter) (string, error) {
	passphrase, err := prompt.InputSecret("", locale.T("previous_password_prompt"))
	if err != nil {
		return "", locale.WrapInputError(err, "auth_err_password_prompt")
	}
	return passphrase, nil
}

func promptUserToRegenerateKeypair(passphrase string, out output.Outputer, prompt prompt.Prompter) error {
	var err error
	// previous passphrase is invalid, inform user and ask if they want to generate a new keypair
	out.Notice(locale.T("auth_generate_new_keypair_message"))
	yes, err := prompt.Confirm("", locale.T("auth_confirm_generate_new_keypair_prompt"), new(bool))
	if err != nil {
		return err
	}
	if yes {
		_, err = keypairs.GenerateAndSaveEncodedKeypair(secretsapi.Get(), passphrase, constants.DefaultRSABitLength)
		// TODO delete user's secrets
	} else {
		err = locale.NewError("auth_err_unrecoverable_keypair")
	}
	return err
}
