package keypairs

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/keys"
	secretModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchRaw fetchs the current user's encoded and unparsed keypair or returns a failure.
func FetchRaw(secretsClient *secretsapi.Client) (*secretModels.Keypair, error) {
	kpOk, err := secretsClient.Keys.GetKeypair(nil, authentication.Get().ClientAuth())
	if err != nil {
		if api.ErrorCode(err) == 404 {
			return nil, secretsapi.FailKeypairNotFound.New("keypair_err_not_found")
		}
		logging.Error("Error when fetching keypair: %v", api.ErrorMessageFromPayload(err))
		return nil, api.FailUnknown.Wrap(err)
	}

	return kpOk.Payload, nil
}

// Fetch fetchs and parses the current user's keypair using the provided passphrase or returns a failure.
func Fetch(secretsClient *secretsapi.Client, passphrase string) (Keypair, error) {
	rawKP, failure := FetchRaw(secretsClient)
	if failure != nil {
		return nil, failure
	}

	kp, failure := ParseEncryptedRSA(*rawKP.EncryptedPrivateKey, passphrase)
	if failure != nil {
		return nil, failure
	}

	return kp, nil
}

// FetchPublicKey fetchs the PublicKey for a sepcific user.
func FetchPublicKey(secretsClient *secretsapi.Client, user *mono_models.User) (Encrypter, error) {
	params := keys.NewGetPublicKeyParams()
	params.UserID = user.UserID
	pubKeyOk, err := secretsClient.Keys.GetPublicKey(params, authentication.Get().ClientAuth())
	if err != nil {
		if api.ErrorCode(err) == 404 {
			return nil, secretsapi.FailPublicKeyNotFound.New("keypair_err_publickey_not_found", user.Username, user.UserID.String())
		}
		return nil, api.FailUnknown.Wrap(err)
	}

	pubKey, failure := ParseRSAPublicKey(*pubKeyOk.Payload.Value)
	if failure != nil {
		return nil, failure
	}

	return pubKey, nil
}
