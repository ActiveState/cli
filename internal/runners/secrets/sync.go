package secrets

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsapiClient "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type syncPrimeable interface {
	primer.Projecter
}

type SyncRunParams struct {
}

type Sync struct {
	secretsClient *secretsapi.Client
	proj          *project.Project
}

func NewSync(client *secretsapi.Client, p syncPrimeable) *Sync {
	return &Sync{
		secretsClient: client,
		proj:          p.Project(),
	}
}

func (s *Sync) Run(params SyncRunParams) error {
	if err := checkSecretsAccess(s.proj); err != nil {
		return err
	}

	org, failure := model.FetchOrgByURLName(s.proj.Owner())

	if failure == nil {
		failure = synchronizeEachOrgMember(s.secretsClient, org)
	}

	if failure != nil {
		failure.WithDescription(locale.T("secrets_err"))
	}

	return nil
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

	fmt.Fprint(os.Stdout, locale.Tr("secrets_sync_results_message", strconv.Itoa(updatedCtr), org.Name))
	return nil
}
