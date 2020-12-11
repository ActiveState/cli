package project

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/access"
	"github.com/ActiveState/cli/internal/failures"
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

// FailExpandNoProjectDefined is used when error arises from an expander function being called without a project
var FailExpandNoProjectDefined = failures.Type("project.fail.secrets.expand.noproject")

// FailInputSecretValue is used when error arises from user providing a secret value.
var FailInputSecretValue = failures.Type("project.fail.secrets.input.value", failures.FailUserInput)

// FailExpandNoAccess is used when the currently authorized user does not have access to project secrets
var FailExpandNoAccess = failures.Type("project.fail.secrets.expand.noaccess", failures.FailUser)

// FailNotAuthenticated is used when trying to access secrets while not being authenticated
var FailNotAuthenticated = failures.Type("project.fail.secrets.noauth", failures.FailUserInput)

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
	secrets         []*secretsModels.UserSecret
	secretsAccessed []*SecretAccess
	cachedSecrets   map[string]string
}

// NewSecretExpander returns a new instance of SecretExpander
func NewSecretExpander(secretsClient *secretsapi.Client, prj *Project, prompt prompt.Prompter) *SecretExpander {
	return &SecretExpander{
		secretsClient: secretsClient,
		cachedSecrets: map[string]string{},
		project:       prj,
		prompt:        prompt,
	}
}

// NewSecretQuietExpander creates an Expander which can retrieve and decrypt stored user secrets.
func NewSecretQuietExpander(secretsClient *secretsapi.Client) ExpanderFunc {
	secretsExpander := NewSecretExpander(secretsClient, nil, nil)
	return secretsExpander.Expand
}

// NewSecretPromptingExpander creates an Expander which can retrieve and decrypt stored user secrets. Additionally,
// it will prompt the user to provide a value for a secret -- in the event none is found -- and save the new
// value with the secrets service.
func NewSecretPromptingExpander(secretsClient *secretsapi.Client, prompt prompt.Prompter) ExpanderFunc {
	secretsExpander := NewSecretExpander(secretsClient, nil, prompt)
	return secretsExpander.ExpandWithPrompt
}

// KeyPair acts as a caching layer for secrets.LoadKeypairFromConfigDir, and ensures that we have a projectfile
func (e *SecretExpander) KeyPair() (keypairs.Keypair, *failures.Failure) {
	if e.projectFile == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}

	if !authentication.Get().Authenticated() {
		return nil, FailNotAuthenticated.New(locale.T("secrets_err_not_authenticated"))
	}

	var fail *failures.Failure
	if e.keypair == nil {
		e.keypair, fail = secrets.LoadKeypairFromConfigDir()
		if fail != nil {
			return nil, fail
		}
	}

	return e.keypair, nil
}

// Organization acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Organization() (*mono_models.Organization, *failures.Failure) {
	if e.project == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}
	var fail *failures.Failure
	if e.organization == nil {
		e.organization, fail = model.FetchOrgByURLName(e.project.Owner())
		if fail != nil {
			return nil, fail
		}
	}

	return e.organization, nil
}

// Project acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Project() (*mono_models.Project, *failures.Failure) {
	if e.project == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}
	var fail *failures.Failure
	if e.remoteProject == nil {
		e.remoteProject, fail = model.FetchProjectByName(e.project.Owner(), e.project.Name())
		if fail != nil {
			return nil, fail
		}
	}

	return e.remoteProject, nil
}

// Secrets acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Secrets() ([]*secretsModels.UserSecret, *failures.Failure) {
	if e.secrets == nil {
		org, fail := e.Organization()
		if fail != nil {
			return nil, fail
		}

		e.secrets, fail = secretsapi.FetchAll(e.secretsClient, org)
		if fail != nil {
			return nil, fail
		}
	}

	return e.secrets, nil
}

// FetchSecret retrieves the given secret
func (e *SecretExpander) FetchSecret(name string, isUser bool) (string, *failures.Failure) {
	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", nil
	}

	userSecret, fail := e.FindSecret(name, isUser)
	if fail != nil {
		return "", fail
	}
	if userSecret == nil {
		return "", secretsapi.FailUserSecretNotFound.New("secrets_expand_err_not_found", name)
	}

	decrBytes, fail := keypair.DecodeAndDecrypt(*userSecret.Value)
	if fail != nil {
		return "", fail
	}

	return string(decrBytes), nil
}

// FetchDefinition retrieves the definition associated with a secret
func (e *SecretExpander) FetchDefinition(name string, isUser bool) (*secretsModels.SecretDefinition, *failures.Failure) {
	defs, fail := secretsapi.FetchDefinitions(e.secretsClient, e.remoteProject.ProjectID)
	if fail != nil {
		return nil, fail
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
func (e *SecretExpander) FindSecret(name string, isUser bool) (*secretsModels.UserSecret, *failures.Failure) {
	owner := e.project.Owner()
	allowed, fail := access.Secrets(owner)
	if fail != nil {
		return nil, fail
	}
	if !allowed {
		return nil, FailExpandNoAccess.New("secrets_expand_err_no_access", owner)
	}

	secrets, fail := e.Secrets()
	if fail != nil {
		return nil, fail
	}

	project, fail := e.Project()
	if fail != nil {
		return nil, fail
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
type SecretFunc func(name string, project *Project) (string, *failures.Failure)

var ErrSecretNotFound = errors.New("secret not found")

// Expand will expand a variable to a secret value, if no secret exists it will return an empty string
func (e *SecretExpander) Expand(_ string, category string, name string, isFunction bool, project *Project) (string, error) {
	isUser := category == UserCategory

	if e.project == nil {
		e.project = project
	}
	if e.projectFile == nil {
		e.projectFile = project.Source()
	}

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", fail.ToError()
	}

	if knownValue, exists := e.cachedSecrets[category+name]; exists {
		return knownValue, nil
	}

	userSecret, fail := e.FindSecret(name, isUser)
	if fail != nil {
		return "", fail.ToError()
	}

	if userSecret == nil {
		return "", locale.WrapInputError(ErrSecretNotFound, "secrets_expand_err_not_found", "Unable to obtain value for secret: `{{.V0}}.`", name)
	}

	decrBytes, fail := keypair.DecodeAndDecrypt(*userSecret.Value)
	if fail != nil {
		return "", fail
	}

	secretValue := string(decrBytes)
	e.cachedSecrets[category+name] = secretValue
	return secretValue, nil
}

// ExpandWithPrompt will expand a variable to a secret value, if no secret exists the user will be prompted
func (e *SecretExpander) ExpandWithPrompt(_ string, category string, name string, isFunction bool, project *Project) (string, error) {
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

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", fail.ToError()
	}

	value, fail := e.FetchSecret(name, isUser)
	if fail != nil && !fail.Type.Matches(secretsapi.FailUserSecretNotFound) {
		return "", fail.ToError()
	}

	if fail == nil {
		return value, nil
	}

	def, fail := e.FetchDefinition(name, isUser)
	if fail != nil {
		return "", fail
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
	if value, fail = e.prompt.InputSecret(locale.Tl("secret_expand", "Secret Expansion"), locale.Tr("secret_value_prompt", name)); fail != nil {
		return "", locale.NewInputError("secrets_err_value_prompt", "The provided secret value is invalid.")
	}

	pj, fail := e.Project()
	if fail != nil {
		return "", fail
	}
	org, fail := e.Organization()
	if fail != nil {
		return "", fail
	}

	fail = secrets.Save(e.secretsClient, keypair, org, pj, isUser, name, value)

	if fail != nil {
		return "", fail
	}

	// Cache it so we're not repeatedly prompting for the same secret
	e.cachedSecrets[category+name] = value

	return value, nil
}
