package secrets

import (
	"github.com/ActiveState/cli/internal/keypairs"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
)

// ShareFromDiff decrypts a source user's secrets that they are sharing and re-encrypts those secrets using
// the public key of a target user provided in the UserSecretDiff struct. This is effectively "copying" a set
// of secrets for use by another user.
func ShareFromDiff(sourceKeypair keypairs.Keypair, diff *secretsModels.UserSecretDiff) ([]*secretsModels.UserSecretShare, error) {
	targetPubKey, err := keypairs.ParseRSAPublicKey(*diff.PublicKey)
	if err != nil {
		return nil, err
	}

	targetShares := make([]*secretsModels.UserSecretShare, len(diff.Shares))
	for idx, sourceShare := range diff.Shares {
		decrVal, err := sourceKeypair.DecodeAndDecrypt(*sourceShare.Value)
		if err != nil {
			return nil, err
		}

		targetSecret, err := targetPubKey.EncryptAndEncode(decrVal)
		if err != nil {
			return nil, err
		}

		targetShares[idx] = &secretsModels.UserSecretShare{
			ProjectID: sourceShare.ProjectID,
			Name:      sourceShare.Name,
			Value:     &targetSecret,
		}
	}
	return targetShares, nil
}
