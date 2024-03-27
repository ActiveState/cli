package secrets

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type setPrimeable interface {
	primer.Projecter
	primer.Configurer
	primer.Auther
	primer.Outputer
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
	auth *authentication.Auth
	out  output.Outputer
}

// NewSet prepares a set execution context for use.
func NewSet(p setPrimeable) *Set {
	return &Set{
		proj: p.Project(),
		cfg:  p.Config(),
		auth: p.Auth(),
		out:  p.Output(),
	}
}

// Run executes the set behavior.
func (s *Set) Run(params SetRunParams) error {
	s.out.Notice(locale.Tr("operating_message", s.proj.NamespaceString(), s.proj.Dir()))
	if err := checkSecretsAccess(s.proj, s.auth); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	secret, err := getSecret(s.proj, params.Name, s.cfg, s.auth)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	org, err := model.FetchOrgByURLName(s.proj.Owner(), s.auth)
	if err != nil {
		return err
	}

	remoteProject, err := model.LegacyFetchProjectByName(org.URLname, s.proj.Name())
	if err != nil {
		return err
	}

	kp, err := secrets.LoadKeypairFromConfigDir(s.cfg)
	if err != nil {
		return err
	}

	err = secrets.Save(secretsapi.GetClient(s.auth), kp, org, remoteProject, secret.IsUser(), secret.Name(), params.Value, s.auth)
	if err != nil {
		return err
	}

	if secret.IsProject() {
		return secrets.ShareWithOrgUsers(secretsapi.GetClient(s.auth), org, remoteProject, secret.Name(), params.Value, s.auth)
	}

	return nil
}
