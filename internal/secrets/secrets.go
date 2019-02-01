package secrets

import (
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/organizations"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsapiClient "github.com/ActiveState/cli/internal/secrets-api/client/secrets"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
)

// Save will add a new secret for this user or update an existing one.
func Save(secretsClient *secretsapi.Client, encrypter keypairs.Encrypter, org *models.Organization, project *models.Project,
	isUser bool, secretName, secretValue string) *failures.Failure {

	logging.Debug("attempting to upsert user-secret for org=%s", org.OrganizationID.String())
	encrStr, failure := encrypter.EncryptAndEncode([]byte(secretValue))
	if failure != nil {
		return failure
	}

	params := secretsapiClient.NewSaveAllUserSecretsParams()
	params.OrganizationID = org.OrganizationID
	secretChange := &secretsModels.UserSecretChange{
		Name:   &secretName,
		Value:  &encrStr,
		IsUser: &isUser,
	}
	if project != nil {
		secretChange.ProjectID = project.ProjectID
	}

	params.UserSecrets = append(params.UserSecrets, secretChange)

	_, err := secretsClient.Secrets.Secrets.SaveAllUserSecrets(params, secretsClient.Auth)
	if err != nil {
		logging.Error("error saving user secret: %v", err)
		return secretsapi.FailSave.New("variables_err_save")
	}

	return nil
}

// ShareWithOrgUsers will share the provided secret with all other users in the organization
// who have a valid public-key available.
func ShareWithOrgUsers(secretsClient *secretsapi.Client, org *models.Organization, project *models.Project, secretName, secretValue string) *failures.Failure {
	currentUserID, failure := secretsClient.AuthenticatedUserID()
	if failure != nil {
		return failure
	}

	members, failure := organizations.FetchMembers(org.Urlname)
	if failure != nil {
		return failure
	}

	for _, member := range members {
		if currentUserID != member.User.UserID {
			pubKey, failure := keypairs.FetchPublicKey(secretsClient, member.User)
			if failure != nil {
				if failure.Type.Matches(secretsapi.FailNotFound) {
					logging.Info("User `%s` has no public-key", member.User.Username)
					// this is okay, just do what we can
					continue
				}
				return failure
			}

			ciphertext, failure := pubKey.EncryptAndEncode([]byte(secretValue))
			if failure != nil {
				logging.Error("Encryptying secret `%s` for user `%s`: %s", secretName, member.User.Username)
				// this is a local issue with the user's keys, so we try and move on
				continue
			}

			share := &secretsModels.UserSecretShare{
				Name:  &secretName,
				Value: &ciphertext,
			}
			if project != nil {
				share.ProjectID = project.ProjectID
			}

			failure = secretsapi.SaveSecretShares(secretsClient, org, member.User, []*secretsModels.UserSecretShare{share})
			if failure != nil {
				// a potentially unrecoverable failure, so we stop here
				return failure
			}
			logging.Info("Update secret `%s` for user `%s`", secretName, member.User.Username)
		}
	}

	return nil
}

func LoadKeypairFromConfigDir() (keypairs.Keypair, *failures.Failure) {
	kp, failure := keypairs.LoadWithDefaults()
	if failure != nil {
		if failure.Type.Matches(keypairs.FailLoadNotFound) || failure.Type.Matches(keypairs.FailKeypairParse) {
			logging.Error("failure loading local keypair: %v", failure)
			return nil, failure.Type.New("keypair_err_require_auth")
		}
		return nil, failure
	}
	return kp, nil
}
