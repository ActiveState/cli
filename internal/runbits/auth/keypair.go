package auth

import (
	"errors"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// ensureUserKeypair checks to see if the currently authenticated user has a Keypair. If not, one is generated
// and saved.
func ensureUserKeypair(passphrase string, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	keypairRes, err := keypairs.FetchRaw(secretsapi.Get(auth), cfg, auth)
	var errKeypairNotFound *keypairs.ErrKeypairNotFound
	if err == nil {
		err = processExistingKeypairForUser(keypairRes, passphrase, cfg, out, prompt, auth)
	} else if errors.As(err, &errKeypairNotFound) {
		err = generateKeypairForUser(cfg, passphrase, auth)
	}

	if err != nil {
		return locale.WrapError(err, "err_ensure_keypair", "Could not find keypair. Please login with '[ACTIONABLE]state auth --prompt[/RESET]'.")
	}

	return nil
}

// generateKeypairForUser attempts to generate and save a Keypair for the currently authenticated user.
func generateKeypairForUser(cfg keypairs.Configurable, passphrase string, auth *authentication.Auth) error {
	_, err := keypairs.GenerateAndSaveEncodedKeypair(cfg, secretsapi.Get(auth), passphrase, constants.DefaultRSABitLength, auth)
	if err != nil {
		return errs.Wrap(err, "Could not generate and save encoded keypair.")
	}
	return nil
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
func processExistingKeypairForUser(keypairRes *secretsModels.Keypair, passphrase string, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	keypair, err := keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, passphrase)
	var errKeypairPassphrase *keypairs.ErrKeypairPassphrase
	if err == nil {
		// yay, store keypair locally just in case it isn't
		return keypairs.SaveWithDefaults(cfg, keypair)
	} else if !errors.As(err, &errKeypairPassphrase) {
		// err did not involve an unmatched passphrase
		return err
	}

	// failed to decrypt stored private-key with provided passphrase, try using a local private-key
	var localKeypair keypairs.Keypair
	localKeypair, err = keypairs.LoadWithDefaults(cfg)
	if err == nil && localKeypair.MatchPublicKey(*keypairRes.PublicKey) {
		// locally stored private-key has a matching public-key, encrypt that with new passphrase and upload
		var encodedKeypair *keypairs.EncodedKeypair
		if encodedKeypair, err = keypairs.EncodeKeypair(localKeypair, passphrase); err != nil {
			return err
		}
		return keypairs.SaveEncodedKeypair(cfg, secretsapi.Get(auth), encodedKeypair, auth)
	}

	// failed to validate with local private-key, try using previous passphrase
	err = recoverKeypairFromPreviousPassphrase(keypairRes, passphrase, cfg, out, prompt, auth)
	if err != nil && errors.As(err, &errKeypairPassphrase) {
		// that failed, see if they want to regenerate their passphrase
		err = promptUserToRegenerateKeypair(passphrase, cfg, out, prompt, auth)
	}
	return err
}

func recoverKeypairFromPreviousPassphrase(keypairRes *secretsModels.Keypair, passphrase string, cfg keypairs.Configurable, out output.Outputer, prompt prompt.Prompter, auth *authentication.Auth) error {
	out.Notice(locale.T("previous_password_message"))
	prevPassphrase, err := promptForPreviousPassphrase(prompt)
	if err == nil {
		var keypair keypairs.Keypair
		keypair, err = keypairs.ParseEncryptedRSA(*keypairRes.EncryptedPrivateKey, prevPassphrase)
		if err == nil {
			// previous passphrase is valid, encrypt private-key with new passphrase and upload
			encodedKeypair, err := keypairs.EncodeKeypair(keypair, passphrase)
			if err == nil {
				if saveErr := keypairs.SaveEncodedKeypair(cfg, secretsapi.Get(auth), encodedKeypair, auth); saveErr != nil {
					return saveErr
				}
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

func promptUserToRegenerateKeypair(passphrase string, cfg keypairs.Configurable, out output.Outputer, prmpt prompt.Prompter, auth *authentication.Auth) error {
	// previous passphrase is invalid, inform user and ask if they want to generate a new keypair
	out.Notice(locale.T("auth_generate_new_keypair_message"))
	yes, kind, err := prmpt.Confirm("", locale.T("auth_confirm_generate_new_keypair_prompt"), ptr.To(false), nil)
	if err != nil {
		return errs.Wrap(err, "Unable to confirm")
	}
	if !yes {
		if kind == prompt.NonInteractive {
			return locale.NewInputError("prompt_abort_non_interactive")
		}
		return locale.NewInputError("auth_err_unrecoverable_keypair")
	}
	_, err = keypairs.GenerateAndSaveEncodedKeypair(cfg, secretsapi.Get(auth), passphrase, constants.DefaultRSABitLength, auth)
	// TODO delete user's secrets
	return err
}
