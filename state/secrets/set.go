package secrets

import (
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/client/secrets"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
)

func buildSetCommand(cmd *Command) *commands.Command {
	return &commands.Command{
		Name:        "set",
		Description: "secrets_set_cmd_description",
		Run:         cmd.ExecuteSet,

		Flags: []*commands.Flag{
			&commands.Flag{
				Name:        "project",
				Shorthand:   "p",
				Description: "secrets_set_flag_project",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.IsProject,
			},
			&commands.Flag{
				Name:        "user",
				Shorthand:   "u",
				Description: "secrets_set_flag_user",
				Type:        commands.TypeBool,
				BoolVar:     &cmd.Flags.IsUser,
			},
		},

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "secrets_set_arg_name_name",
				Description: "secrets_set_arg_name_description",
				Variable:    &cmd.Args.SecretName,
				Required:    true,
			},
			&commands.Argument{
				Name:        "secrets_set_arg_value_name",
				Description: "secrets_set_arg_value_description",
				Variable:    &cmd.Args.SecretValue,
				Required:    true,
			},
		},
	}
}

// ExecuteSet processes the `secrets set` command.
func (cmd *Command) ExecuteSet(_ *cobra.Command, args []string) {
	projectFile := projectfile.Get()
	org, failure := organizations.FetchByURLName(projectFile.Owner)
	if failure == nil {
		var project *models.Project
		var kp keypairs.Keypair
		if cmd.Flags.IsProject {
			project, failure = projects.FetchByName(org.Urlname, projectFile.Name)
		}

		if failure == nil {
			kp, failure = loadKeypairFromConfigDir()
		}

		if failure == nil {
			failure = saveUserSecret(cmd.secretsClient, kp, org, project, cmd.Flags.IsUser, cmd.Args.SecretName, cmd.Args.SecretValue)
		}

		if failure == nil && !cmd.Flags.IsUser {
			failure = shareSecretWithOrgUsers(cmd.secretsClient, org, project, cmd.Args.SecretName, cmd.Args.SecretValue)
		}
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}
}

// saveUserSecret will add a new secret for this user or update an existing one.
func saveUserSecret(secretsClient *secretsapi.Client, encrypter keypairs.Encrypter, org *models.Organization, project *models.Project,
	isUser bool, secretName, secretValue string) *failures.Failure {

	logging.Debug("attempting to upsert user-secret for org=%s", org.OrganizationID.String())
	encrStr, failure := encrypter.EncryptAndEncode([]byte(secretValue))
	if failure != nil {
		return failure
	}

	params := secrets.NewSaveAllUserSecretsParams()
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
		return secretsapi.FailSave.New("secrets_err_save")
	}

	return nil
}

// shareSecretWithOrgUsers will share the provided secret with all other users in the organization
// who have a valid public-key available.
func shareSecretWithOrgUsers(secretsClient *secretsapi.Client, org *models.Organization, project *models.Project, secretName, secretValue string) *failures.Failure {
	currentUserID, failure := secretsClient.Authenticated()
	if failure != nil {
		return failure
	}

	members, failure := organizations.FetchMembers(org.Urlname)
	if failure != nil {
		return failure
	}

	for _, member := range members {
		if *currentUserID != member.User.UserID {
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

			failure = saveUserSecretShares(secretsClient, org, member.User, []*secretsModels.UserSecretShare{share})
			if failure != nil {
				// a potentially unrecoverable failure, so we stop here
				return failure
			}
			logging.Info("Update secret `%s` for user `%s`", secretName, member.User.Username)
		}
	}

	return nil
}
