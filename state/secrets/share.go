package secrets

import (
	"strings"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/organizations"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/secrets"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
)

func buildShareCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "share",
		Description: "secrets_share_cmd_description",
		Run:         cmd.ExecuteShare,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "secrets_share_arg_user_name",
				Description: "secrets_share_arg_user_description",
				Variable:    &cmd.Args.ShareUserHandle,
				Required:    true,
			},
		},
	}
}

// ExecuteShare processes the `secrets share` command.
func (cmd *Command) ExecuteShare(_ *cobra.Command, args []string) {
	projectFile := projectfile.Get()
	org, failure := organizations.FetchByURLName(projectFile.Owner)
	if failure == nil {
		var member *models.Member
		member, failure = findMemberForOrgByUsername(org, cmd.Args.ShareUserHandle)
		if failure == nil {
			failure = shareSecrets(cmd.secretsClient, org, member.User)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}
}

func findMemberForOrgByUsername(org *models.Organization, userHandler string) (*models.Member, *failures.Failure) {
	members, failure := organizations.FetchMembers(org.Urlname)
	if failure != nil {
		return nil, failure
	}

	for _, member := range members {
		if strings.EqualFold(userHandler, member.User.Username) {
			return member, nil
		}
	}
	return nil, api.FailNotFound.New("err_api_member_not_found")
}

func shareSecrets(secretsClient *secretsapi.Client, org *models.Organization, forUser *models.User) *failures.Failure {
	selfSecrets, failure := fetchAll(secretsClient, org)
	if failure != nil {
		return failure
	}

	otherEncrypter, failure := keypairs.FetchPublicKey(secretsClient, forUser)
	if failure != nil {
		return failure
	}

	selfKeypair, failure := keypairs.Fetch(secretsClient)
	if failure != nil {
		return failure
	}

	otherChanges, failure := portShareableSecrets(selfSecrets, selfKeypair, otherEncrypter)

	return saveOtherUserSecrets(secretsClient, org, forUser, otherChanges)
}

func portShareableSecrets(selfSecrets []*secretsModels.UserSecret, decrypter keypairs.Decrypter, encrypter keypairs.Encrypter) ([]*secretsModels.UserSecretChange, *failures.Failure) {
	var otherSecrets []*secretsModels.UserSecretChange

	for _, selfSecret := range selfSecrets {
		if !*selfSecret.IsUser {
			plaintextValue, failure := decrypter.DecodeAndDecrypt(*selfSecret.Value)
			if failure != nil {
				return nil, failure
			}

			ciphertext, failure := encrypter.EncryptAndEncode(plaintextValue)
			if failure != nil {
				return nil, failure
			}

			otherSecrets = append(otherSecrets, &secretsModels.UserSecretChange{
				ProjectID: selfSecret.ProjectID,
				Name:      selfSecret.Name,
				IsUser:    selfSecret.IsUser,
				Value:     &ciphertext,
			})
		}
	}

	return otherSecrets, nil
}

func saveOtherUserSecrets(secretsClient *secretsapi.Client, org *models.Organization, user *models.User, changes []*secretsModels.UserSecretChange) *failures.Failure {
	params := secrets.NewSaveOtherUserSecretsParams()
	params.OrganizationID = org.OrganizationID
	params.UserID = user.UserID
	params.UserSecrets = changes
	_, err := secretsClient.Secrets.Secrets.SaveOtherUserSecrets(params, secretsClient.Auth)
	if err != nil {
		logging.Debug("error sharing user secrets: %v", err)
		return secretsapi.FailSave.New("secrets_err_save")
	}
	return nil
}
