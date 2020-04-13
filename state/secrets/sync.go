package secrets

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsapiClient "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
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
	org, failure := model.FetchOrgByURLName(project.Owner())

	if failure == nil {
		failure = synchronizeEachOrgMember(cmd.secretsClient, org)
	}

	if failure != nil {
		failures.Handle(failure, locale.T("secrets_err"))
	}
}

func synchronizeEachOrgMember(secretsClient *secretsapi.Client, org *mono_models.Organization) *failures.Failure {
	sourceKeypair, failure := secrets.LoadKeypairFromConfigDir()
	if failure != nil {
		return failure
	}

	members, failure := model.FetchOrgMembers(org.URLname)
	if failure != nil {
		return failure
	}

	currentUserID, failure := secretsClient.AuthenticatedUserID()
	if failure != nil {
		return failure
	}

	updatedCtr := int(0)
	for _, member := range members {
		if currentUserID != member.User.UserID {
			params := secretsapiClient.NewDiffUserSecretsParams()
			params.OrganizationID = org.OrganizationID
			params.UserID = member.User.UserID
			diffPayloadOk, err := secretsClient.Secrets.Secrets.DiffUserSecrets(params, authentication.Get().ClientAuth())

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

			targetShares, fail := secrets.ShareFromDiff(sourceKeypair, diffPayloadOk.Payload)
			if fail != nil {
				return fail
			}

			fail = secretsapi.SaveSecretShares(secretsClient, org, member.User, targetShares)
			if fail != nil {
				return fail
			}
			updatedCtr++
		}
	}

	print.Line(locale.Tr("secrets_sync_results_message", strconv.Itoa(updatedCtr), org.Name))
	return nil
}
