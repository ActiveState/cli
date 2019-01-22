package secrets

import (
	"strconv"

	"github.com/ActiveState/cli/internal/keypairs"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/print"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/secrets"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

func buildSyncCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "sync",
		Description: "secrets_sync_cmd_description",
		Run:         cmd.ExecuteSync,
	}
}

// ExecuteSync processes the `secrets sync` command.
func (cmd *Command) ExecuteSync(_ *cobra.Command, args []string) {
	project := project.Get()
	org, failure := organizations.FetchByURLName(project.Owner())

	if failure == nil {
		failure = synchronizeEachOrgMember(cmd.secretsClient, org)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}
}

func synchronizeEachOrgMember(secretsClient *secretsapi.Client, org *models.Organization) *failures.Failure {
	sourceKeypair, failure := loadKeypairFromConfigDir()
	if failure != nil {
		return failure
	}

	members, failure := organizations.FetchMembers(org.Urlname)
	if failure != nil {
		return failure
	}

	currentUserID, failure := secretsClient.Authenticated()
	if failure != nil {
		return failure
	}

	updatedCtr := int(0)
	for _, member := range members {
		if *currentUserID != member.User.UserID {
			params := secrets.NewDiffUserSecretsParams()
			params.OrganizationID = org.OrganizationID
			params.UserID = member.User.UserID
			diffPayloadOk, err := secretsClient.Secrets.Secrets.DiffUserSecrets(params, secretsClient.Auth)

			if err != nil {
				switch statusCode := api.ErrorCode(err); statusCode {
				case 404:
					continue // nothing to do when no diff for a user, move on to next one
				case 401:
					return api.FailAuth.New("err_api_not_authenticated")
				default:
					logging.Debug("unknown error diffing user secrets with %s: %v", member.User.UserID.String(), err)
					return api.FailUnknown.Wrap(err)
				}
			}

			targetShares, failure := processDiffShares(sourceKeypair, diffPayloadOk.Payload)
			if failure != nil {
				return failure
			}

			failure = saveUserSecretShares(secretsClient, org, member.User, targetShares)
			if failure != nil {
				return failure
			}
			updatedCtr++
		}
	}

	print.Line(locale.Tr("secrets_sync_result_message", strconv.Itoa(updatedCtr), org.Name))
	return nil
}

// processDiffShares decrypts a source user's secrets that they are sharing and re-encrypts those secrets using
// the public key of a target user provided in the UserSecretDiff struct. This is effectively "copying" a set
// of secrets for use by another user.
func processDiffShares(sourceKeypair keypairs.Keypair, diff *secretsModels.UserSecretDiff) ([]*secretsModels.UserSecretShare, *failures.Failure) {
	targetPubKey, failure := keypairs.ParseRSAPublicKey(*diff.PublicKey)
	if failure != nil {
		return nil, failure
	}

	targetShares := make([]*secretsModels.UserSecretShare, len(diff.Shares))
	for idx, sourceShare := range diff.Shares {
		decrVal, failure := sourceKeypair.DecodeAndDecrypt(*sourceShare.Value)
		if failure != nil {
			return nil, failure
		}

		targetSecret, failure := targetPubKey.EncryptAndEncode(decrVal)
		if failure != nil {
			return nil, failure
		}

		targetShares[idx] = &secretsModels.UserSecretShare{
			ProjectID: sourceShare.ProjectID,
			Name:      sourceShare.Name,
			Value:     &targetSecret,
		}
	}
	return targetShares, nil
}
