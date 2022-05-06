package project

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/access"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// UserCategory is the string used when referencing user secrets (eg. $secrets.user.foo)
const UserCategory = "user"

// ProjectCategory is the string used when referencing project secrets (eg. $secrets.project.foo)
const ProjectCategory = "project"

// SecretAccess is used to track secrets that were requested
type SecretAccess struct {
	IsUser bool
	Name   string
}

// SecretExpander takes care of expanding secrets
type SecretExpander struct {
	secretsClient   *secretsapi.Client
	keypair         keypairs.Keypair
	organization    *mono_models.Organization
	remoteProject   *mono_models.Project
	projectFile     *projectfile.Project
	project         *Project
	prompt          prompt.Prompter
	cfg             keypairs.Configurable
	secrets         []*secretsModels.UserSecret
	secretsAccessed []*SecretAccess
	cachedSecrets   map[string]string
}

// NewSecretExpander returns a new instance of SecretExpander
func NewSecretExpander(secretsClient *secretsapi.Client, prj *Project, prompt prompt.Prompter, cfg keypairs.Configurable) *SecretExpander {
	return &SecretExpander{
		secretsClient: secretsClient,
		cachedSecrets: map[string]string{},
		project:       prj,
		prompt:        prompt,
		cfg:           cfg,
	}
}

// NewSecretQuietExpander creates an Expander which can retrieve and decrypt stored user secrets.
func NewSecretQuietExpander(secretsClient *secretsapi.Client, cfg keypairs.Configurable) ExpanderFunc {
	secretsExpander := NewSecretExpander(secretsClient, nil, nil, cfg)
	return secretsExpander.Expand
}

// NewSecretPromptingExpander creates an Expander which can retrieve and decrypt stored user secrets. Additionally,
// it will prompt the user to provide a value for a secret -- in the event none is found -- and save the new
// value with the secrets service.
func NewSecretPromptingExpander(secretsClient *secretsapi.Client, prompt prompt.Prompter, cfg keypairs.Configurable) ExpanderFunc {
	secretsExpander := NewSecretExpander(secretsClient, nil, prompt, cfg)
	return secretsExpander.ExpandWithPrompt
}

// KeyPair acts as a caching layer for secrets.LoadKeypairFromConfigDir, and ensures that we have a projectfile
func (e *SecretExpander) KeyPair() (keypairs.Keypair, error) {
	if e.projectFile == nil {
		return nil, locale.NewError("secrets_err_expand_noproject")
	}

	if !authentication.LegacyGet().Authenticated() {
		return nil, locale.NewInputError("secrets_err_not_authenticated")
	}

	var err error
	if e.keypair == nil {
		e.keypair, err = secrets.LoadKeypairFromConfigDir(e.cfg)
		if err != nil {
			return nil, err
		}
	}

	return e.keypair, nil
}

// Organization acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Organization() (*mono_models.Organization, error) {
	if e.project == nil {
		return nil, locale.NewError("secrets_err_expand_noproject")
	}
	var err error
	if e.organization == nil {
		e.organization, err = model.FetchOrgByURLName(e.project.Owner())
		if err != nil {
			return nil, err
		}
	}

	return e.organization, nil
}

// Project acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Project() (*mono_models.Project, error) {
	if e.project == nil {
		return nil, locale.NewError("secrets_err_expand_noproject")
	}
	var err error
	if e.remoteProject == nil {
		e.remoteProject, err = model.FetchProjectByName(e.project.Owner(), e.project.Name())
		if err != nil {
			return nil, err
		}
	}

	return e.remoteProject, nil
}

// Secrets acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Secrets() ([]*secretsModels.UserSecret, error) {
	if e.secrets == nil {
		org, err := e.Organization()
		if err != nil {
			return nil, err
		}

		e.secrets, err = secretsapi.FetchAll(e.secretsClient, org)
		if err != nil {
			return nil, err
		}
	}

	return e.secrets, nil
}

// FetchSecret retrieves the given secret
func (e *SecretExpander) FetchSecret(name string, isUser bool) (string, error) {
	keypair, err := e.KeyPair()
	if err != nil {
		return "", nil
	}

	userSecret, err := e.FindSecret(name, isUser)
	if err != nil {
		return "", err
	}
	if userSecret == nil {
		return "", locale.WrapInputError(ErrSecretNotFound, "secrets_expand_err_not_found", "", name)
	}

	decrBytes, err := keypair.DecodeAndDecrypt(*userSecret.Value)
	if err != nil {
		return "", err
	}

	return string(decrBytes), nil
}

// FetchDefinition retrieves the definition associated with a secret
func (e *SecretExpander) FetchDefinition(name string, isUser bool) (*secretsModels.SecretDefinition, error) {
	defs, err := secretsapi.FetchDefinitions(e.secretsClient, e.remoteProject.ProjectID)
	if err != nil {
		return nil, err
	}

	scope := secretsapi.ScopeUser
	if !isUser {
		scope = secretsapi.ScopeProject
	}

	for _, def := range defs {
		if name == *def.Name && string(scope) == *def.Scope {
			return def, nil
		}
	}

	return nil, nil
}

// FindSecret will find the secret appropriate for the current project
func (e *SecretExpander) FindSecret(name string, isUser bool) (*secretsModels.UserSecret, error) {
	owner := e.project.Owner()
	allowed, err := access.Secrets(owner)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, locale.NewInputError("secrets_expand_err_no_access", "", owner)
	}

	secrets, err := e.Secrets()
	if err != nil {
		return nil, err
	}

	project, err := e.Project()
	if err != nil {
		return nil, err
	}

	e.secretsAccessed = append(e.secretsAccessed, &SecretAccess{isUser, name})

	projectID := project.ProjectID.String()
	variableRequiresUser := isUser
	variableRequiresProject := true

	for _, userSecret := range secrets {
		secretProjectID := userSecret.ProjectID.String()
		secretRequiresUser := userSecret.IsUser != nil && *userSecret.IsUser
		secretRequiresProject := secretProjectID != ""

		nameMatches := strings.EqualFold(*userSecret.Name, name)
		projectMatches := (!variableRequiresProject || secretProjectID == projectID)

		// shareMatches and storeMatches show a detachment from the data due to the secrets-svc api needing a refactor
		// to match the new data structure. Story: https://www.pivotaltracker.com/story/show/166272717
		shareMatches := variableRequiresUser == secretRequiresUser
		storeMatches := variableRequiresProject == secretRequiresProject

		if nameMatches && projectMatches && shareMatches && storeMatches {
			return userSecret, nil
		}
	}

	return nil, nil
}

// SecretsAccessed returns all secrets that were accessed since initialization
func (e *SecretExpander) SecretsAccessed() []*SecretAccess {
	return e.secretsAccessed
}

// SecretFunc defines what our expander functions will be returning
type SecretFunc func(name string, project *Project) (string, error)

var ErrSecretNotFound = errors.New("secret not found")

// Expand will expand a variable to a secret value, if no secret exists it will return an empty string
func (e *SecretExpander) Expand(_ string, category string, name string, isFunction bool, project *Project) (string, error) {
	if !condition.OptInUnstable(e.cfg) {
		return "", locale.NewError("secrets_unstable_warning")
	}

	isUser := category == UserCategory

	if e.project == nil {
		e.project = project
	}
	if e.projectFile == nil {
		e.projectFile = project.Source()
	}

	keypair, err := e.KeyPair()
	if err != nil {
		return "", err
	}

	if knownValue, exists := e.cachedSecrets[category+name]; exists {
		return knownValue, nil
	}

	userSecret, err := e.FindSecret(name, isUser)
	if err != nil {
		return "", err
	}

	if userSecret == nil {
		return "", locale.WrapInputError(ErrSecretNotFound, "secrets_expand_err_not_found", "Unable to obtain value for secret: `{{.V0}}.`", name)
	}

	decrBytes, err := keypair.DecodeAndDecrypt(*userSecret.Value)
	if err != nil {
		return "", err
	}

	secretValue := string(decrBytes)
	e.cachedSecrets[category+name] = secretValue
	return secretValue, nil
}

// ExpandWithPrompt will expand a variable to a secret value, if no secret exists the user will be prompted
func (e *SecretExpander) ExpandWithPrompt(_ string, category string, name string, isFunction bool, project *Project) (string, error) {
	if !condition.OptInUnstable(e.cfg) {
		return "", locale.NewError("secrets_unstable_warning")
	}

	isUser := category == UserCategory

	if knownValue, exists := e.cachedSecrets[category+name]; exists {
		return knownValue, nil
	}

	if e.project == nil {
		e.project = project
	}
	if e.projectFile == nil {
		e.projectFile = project.Source()
	}

	keypair, err := e.KeyPair()
	if err != nil {
		return "", err
	}

	value, err := e.FetchSecret(name, isUser)
	if err != nil && !errors.Is(err, ErrSecretNotFound) {
		return "", err
	}

	if err == nil {
		return value, nil
	}

	def, err := e.FetchDefinition(name, isUser)
	if err != nil {
		return "", err
	}

	scope := string(secretsapi.ScopeUser)
	if !isUser {
		scope = string(secretsapi.ScopeProject)
	}
	description := locale.T("secret_no_description")
	if def != nil && def.Description != "" {
		description = def.Description
	}

	project.Outputer.Notice(locale.Tr("secret_value_prompt_summary", name, description, scope, locale.T("secret_prompt_"+scope)))
	if value, err = e.prompt.InputSecret(locale.Tl("secret_expand", "Secret Expansion"), locale.Tr("secret_value_prompt", name)); err != nil {
		return "", locale.NewInputError("secrets_err_value_prompt", "The provided secret value is invalid.")
	}

	pj, err := e.Project()
	if err != nil {
		return "", err
	}
	org, err := e.Organization()
	if err != nil {
		return "", err
	}

	err = secrets.Save(e.secretsClient, keypair, org, pj, isUser, name, value)

	if err != nil {
		return "", err
	}

	// Cache it so we're not repeatedly prompting for the same secret
	e.cachedSecrets[category+name] = value

	return value, nil
}
