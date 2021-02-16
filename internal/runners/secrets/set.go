package secrets

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type setPrimeable interface {
	primer.Projecter
	primer.Configurer
}

// SetRunParams tracks the info required for running Set.
type SetRunParams struct {
	Name  string
	Value string
}

// Set manages the setting execution context.
type Set struct {
	proj *project.Project
	cfg  keypairs.Configurable
}

// NewSet prepares a set execution context for use.
func NewSet(p setPrimeable) *Set {
	return &Set{
		proj: p.Project(),
		cfg:  p.Config(),
	}
}

// Run executes the set behavior.
func (s *Set) Run(params SetRunParams) error {
	if err := checkSecretsAccess(s.proj); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	secret, err := getSecret(s.proj, params.Name, s.cfg)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	org, err := model.FetchOrgByURLName(s.proj.Owner())
	if err != nil {
		return err
	}

	remoteProject, err := model.FetchProjectByName(org.URLname, s.proj.Name())
	if err != nil {
		return err
	}

	kp, err := secrets.LoadKeypairFromConfigDir(s.cfg)
	if err != nil {
		return err
	}

	err = secrets.Save(secretsapi.GetClient(), kp, org, remoteProject, secret.IsUser(), secret.Name(), params.Value)
	if err != nil {
		return err
	}

	if secret.IsProject() {
		return secrets.ShareWithOrgUsers(secretsapi.GetClient(), org, remoteProject, secret.Name(), params.Value)
	}

	return nil
}
