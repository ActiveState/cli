package keypairs

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/keys"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
)

// SaveEncodedKeypair stores an encoded Keypair back to the Secrets Service.
func SaveEncodedKeypair(secretsClient *secretsapi.Client, encKeypair *EncodedKeypair) *failures.Failure {
	params := keys.NewSaveKeypairParams().WithKeypair(&secretsModels.KeypairChange{
		EncryptedPrivateKey: &encKeypair.EncodedPrivateKey,
		PublicKey:           &encKeypair.EncodedPublicKey,
	})

	if _, err := secretsClient.Keys.SaveKeypair(params, secretsClient.Auth); err != nil {
		return secretsapi.FailKeypairSave.New("keypair_err_save")
	}

	print.Line(locale.T("keypair_generate_success"))

	// save the keypair locally to avoid authenticating the keypair every time it's used
	return SaveWithDefaults(encKeypair.Keypair)
}
