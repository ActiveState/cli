package secrets

import (
	"strconv"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
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
	primer.Outputer
}

// Sync manages the synchronization execution context.
type Sync struct {
	secretsClient *secretsapi.Client
	proj          *project.Project
	out           output.Outputer
}

// NewSync prepares a sync execution context for use.
func NewSync(client *secretsapi.Client, p syncPrimeable) *Sync {
	return &Sync{
		secretsClient: client,
		proj:          p.Project(),
		out:           p.Output(),
	}
}

// Run executes the sync behavior.
func (s *Sync) Run() error {
	if err := checkSecretsAccess(s.proj); err != nil {
		return err
	}

	org, fail := model.FetchOrgByURLName(s.proj.Owner())
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	updatedCount, fail := synchronizeEachOrgMember(s.secretsClient, org)
	if fail != nil {
		return fail.WithDescription(locale.T("secrets_err"))
	}

	s.out.Print(locale.Tr("secrets_sync_results_message", strconv.Itoa(updatedCount), org.Name))

	return nil
}

func synchronizeEachOrgMember(secretsClient *secretsapi.Client, org *mono_models.Organization) (count int, f *failures.Failure) {
	sourceKeypair, fail := secrets.LoadKeypairFromConfigDir()
	if fail != nil {
		return 0, fail
	}

	members, fail := model.FetchOrgMembers(org.URLname)
	if fail != nil {
		return 0, fail
	}

	currentUserID, fail := secretsClient.AuthenticatedUserID()
	if fail != nil {
		return 0, fail
	}

	var updatedCtr int
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
					return updatedCtr, api.FailAuth.New("err_api_not_authenticated")
				default:
					logging.Debug("unknown error diffing user secrets with %s: %v", member.User.UserID.String(), err)
					return updatedCtr, api.FailUnknown.Wrap(err)
				}
			}

			targetShares, fail := secrets.ShareFromDiff(sourceKeypair, diffPayloadOk.Payload)
			if fail != nil {
				return updatedCtr, fail
			}

			fail = secretsapi.SaveSecretShares(secretsClient, org, member.User, targetShares)
			if fail != nil {
				return updatedCtr, fail
			}
			updatedCtr++
		}
	}

	return updatedCtr, nil
}
