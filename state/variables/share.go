package variables

import (
	"github.com/ActiveState/cli/internal/secrets"

	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/organizations"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/cobra"
)

func buildShareCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "share",
		Description: "variables_share_cmd_description",
		Run:         cmd.ExecuteShare,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "variables_share_arg_user_name",
				Description: "variables_share_arg_user_description",
				Variable:    &cmd.Args.ShareUserHandle,
				Required:    true,
			},
		},
	}
}

// ExecuteShare processes the `secrets share` command.
func (cmd *Command) ExecuteShare(_ *cobra.Command, args []string) {
	project := project.Get()
	org, failure := organizations.FetchByURLName(project.Owner())
	if failure == nil {
		var member *models.Member
		member, failure = organizations.FetchMember(org, cmd.Args.ShareUserHandle)
		if failure == nil {
			failure = shareSecrets(cmd.secretsClient, org, member.User)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("variables_err"))
	}
}

func shareSecrets(secretsClient *secretsapi.Client, org *models.Organization, forUser *models.User) *failures.Failure {
	selfSecrets, failure := secretsapi.FetchAll(secretsClient, org)
	if failure != nil {
		return failure
	}

	otherEncrypter, failure := keypairs.FetchPublicKey(secretsClient, forUser)
	if failure != nil {
		return failure
	}

	selfKeypair, failure := secrets.LoadKeypairFromConfigDir()
	if failure != nil {
		return failure
	}

	shares, failure := portShareableSecrets(selfSecrets, selfKeypair, otherEncrypter)
	if failure != nil {
		return failure
	}

	return secretsapi.SaveSecretShares(secretsClient, org, forUser, shares)
}

func portShareableSecrets(selfSecrets []*secretsModels.UserSecret, decrypter keypairs.Decrypter, encrypter keypairs.Encrypter) ([]*secretsModels.UserSecretShare, *failures.Failure) {
	var otherSecrets []*secretsModels.UserSecretShare

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

			otherSecrets = append(otherSecrets, &secretsModels.UserSecretShare{
				ProjectID: selfSecret.ProjectID,
				Name:      selfSecret.Name,
				Value:     &ciphertext,
			})
		}
	}

	return otherSecrets, nil
}
