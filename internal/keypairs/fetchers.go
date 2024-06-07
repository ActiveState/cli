package keypairs

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/keys"
	secretModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type ErrKeypairNotFound struct{ *locale.LocalizedError }

// FetchRaw fetchs the current user keypair or returns a failure.
func FetchRaw(secretsClient *secretsapi.Client, cfg authentication.Configurable, auth *authentication.Auth) (*secretModels.Keypair, error) {
	kpOk, err := secretsClient.Keys.GetKeypair(nil, auth.ClientAuth())
	if err != nil {
		if api.ErrorCode(err) == 404 {
			return nil, &ErrKeypairNotFound{locale.WrapExternalError(err, "keypair_err_not_found")}
		}
		multilog.Error("Error when fetching keypair: %v", api.ErrorMessageFromPayload(err))
		return nil, errs.Wrap(err, "GetKeypair failed")
	}

	return kpOk.Payload, nil
}

// FetchPublicKey fetchs the PublicKey for a sepcific user.
func FetchPublicKey(secretsClient *secretsapi.Client, user *mono_models.User, auth *authentication.Auth) (Encrypter, error) {
	params := keys.NewGetPublicKeyParams()
	params.UserID = user.UserID
	pubKeyOk, err := secretsClient.Keys.GetPublicKey(params, auth.ClientAuth())
	if err != nil {
		if api.ErrorCode(err) == 404 {
			return nil, &ErrKeypairNotFound{locale.WrapExternalError(err, "keypair_err_publickey_not_found", "", user.Username, user.UserID.String())}
		}
		return nil, errs.Wrap(err, "GetPublicKey failed")
	}

	pubKey, err := ParseRSAPublicKey(*pubKeyOk.Payload.Value)
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}
