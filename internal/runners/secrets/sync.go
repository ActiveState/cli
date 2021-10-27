package secrets

import (
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
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
	primer.Configurer
}

// Sync manages the synchronization execution context.
type Sync struct {
	secretsClient *secretsapi.Client
	proj          *project.Project
	out           output.Outputer
	cfg           keypairs.Configurable
}

// NewSync prepares a sync execution context for use.
func NewSync(client *secretsapi.Client, p syncPrimeable) *Sync {
	return &Sync{
		secretsClient: client,
		proj:          p.Project(),
		out:           p.Output(),
		cfg:           p.Config(),
	}
}

// Run executes the sync behavior.
func (s *Sync) Run() error {
	if err := checkSecretsAccess(s.proj); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	org, err := model.FetchOrgByURLName(s.proj.Owner())
	if err != nil {
		return locale.WrapError(err, "secrets_err_fetch_org", "Cannot fetch org")
	}

	updatedCount, err := synchronizeEachOrgMember(s.secretsClient, org, s.cfg)
	if err != nil {
		return locale.WrapError(err, "secrets_err_sync", "Cannot synchronize secrets")
	}

	s.out.Print(locale.Tr("secrets_sync_results_message", strconv.Itoa(updatedCount), org.DisplayName))

	return nil
}

func synchronizeEachOrgMember(secretsClient *secretsapi.Client, org *mono_models.Organization, cfg keypairs.Configurable) (count int, f error) {
	sourceKeypair, err := secrets.LoadKeypairFromConfigDir(cfg)
	if err != nil {
		return 0, err
	}

	members, err := model.FetchOrgMembers(org.URLname)
	if err != nil {
		return 0, err
	}

	currentUserID, err := secretsClient.AuthenticatedUserID()
	if err != nil {
		return 0, err
	}

	var updatedCtr int
	for _, member := range members {
		if currentUserID != member.User.UserID {
			params := secretsapiClient.NewDiffUserSecretsParams()
			params.OrganizationID = org.OrganizationID
			params.UserID = member.User.UserID
			diffPayloadOk, err := secretsClient.Secrets.Secrets.DiffUserSecrets(params, authentication.LegacyGet().ClientAuth())

			if err != nil {
				switch statusCode := api.ErrorCode(err); statusCode {
				case 404:
					continue // nothing to do when no diff for a user, move on to next one
				case 401:
					return updatedCtr, locale.NewInputError("err_api_not_authenticated")
				default:
					logging.Debug("unknown error diffing user secrets with %s: %v", member.User.UserID.String(), err)
					return updatedCtr, errs.Wrap(err, "Unknown failure")
				}
			}

			targetShares, err := secrets.ShareFromDiff(sourceKeypair, diffPayloadOk.Payload)
			if err != nil {
				return updatedCtr, err
			}

			err = secretsapi.SaveSecretShares(secretsClient, org, member.User, targetShares)
			if err != nil {
				return updatedCtr, err
			}
			updatedCtr++
		}
	}

	return updatedCtr, nil
}
