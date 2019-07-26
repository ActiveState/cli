package project

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/secrets"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailExpandNoProjectDefined is used when error arises from an expander function being called without a project
var FailExpandNoProjectDefined = failures.Type("project.fail.secrets.expand.noproject")

// FailInputSecretValue is used when error arises from user providing a secret value.
var FailInputSecretValue = failures.Type("project.fail.secrets.input.value", failures.FailUserInput)

func init() {
	RegisterExpander("secrets.user", NewSecretPromptingExpander(secretsapi.Get(), true))
	RegisterExpander("secrets.project", NewSecretPromptingExpander(secretsapi.Get(), false))

	// Deprecation mechanic
	projectExpander := NewSecretPromptingExpander(secretsapi.Get(), false)
	RegisterExpander("variables", func(name string, project *Project) (string, *failures.Failure) {
		print.Warning(locale.Tr("secrets_warn_deprecated_var_expand", name))
		return projectExpander(name, project)
	})
}

// SecretExpander takes care of expanding secrets
type SecretExpander struct {
	secretsClient *secretsapi.Client
	keypair       keypairs.Keypair
	organization  *mono_models.Organization
	remoteProject *mono_models.Project
	projectFile   *projectfile.Project
	project       *Project
	secrets       []*secretsModels.UserSecret
	cachedSecrets map[string]string
	isUser        bool
}

// NewSecretExpander returns a new instance of SecretExpander
func NewSecretExpander(secretsClient *secretsapi.Client, isUser bool) *SecretExpander {
	return &SecretExpander{
		secretsClient: secretsClient,
		cachedSecrets: map[string]string{},
		isUser:        isUser,
	}
}

// NewSecretQuietExpander creates an Expander which can retrieve and decrypt stored user secrets.
func NewSecretQuietExpander(secretsClient *secretsapi.Client, isUser bool) ExpanderFunc {
	secretsExpander := NewSecretExpander(secretsClient, isUser)
	return secretsExpander.Expand
}

// NewSecretPromptingExpander creates an Expander which can retrieve and decrypt stored user secrets. Additionally,
// it will prompt the user to provide a value for a secret -- in the event none is found -- and save the new
// value with the secrets service.
func NewSecretPromptingExpander(secretsClient *secretsapi.Client, isUser bool) ExpanderFunc {
	secretsExpander := NewSecretExpander(secretsClient, isUser)
	return secretsExpander.ExpandWithPrompt
}

// KeyPair acts as a caching layer for secrets.LoadKeypairFromConfigDir, and ensures that we have a projectfile
func (e *SecretExpander) KeyPair() (keypairs.Keypair, *failures.Failure) {
	if e.projectFile == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
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
	if knownValue, exists := e.cachedSecrets[name]; exists {
		return knownValue, nil
	}

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

	e.cachedSecrets[name] = string(decrBytes)
	return e.cachedSecrets[name], nil
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
	secrets, fail := e.Secrets()
	if fail != nil {
		return nil, fail
	}

	project, fail := e.Project()
	if fail != nil {
		return nil, fail
	}

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

// SecretFunc defines what our expander functions will be returning
type SecretFunc func(name string, project *Project) (string, *failures.Failure)

// Expand will expand a variable to a secret value, if no secret exists it will return an empty string
func (e *SecretExpander) Expand(name string, project *Project) (string, *failures.Failure) {
	if e.project == nil {
		e.project = project
	}
	if e.projectFile == nil {
		e.projectFile = project.Source()
	}

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", fail
	}

	if knownValue, exists := e.cachedSecrets[name]; exists {
		return knownValue, nil
	}

	userSecret, fail := e.FindSecret(name, e.isUser)
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

	secretValue := string(decrBytes)
	e.cachedSecrets[name] = secretValue
	return secretValue, nil
}

// ExpandWithPrompt will expand a variable to a secret value, if no secret exists the user will be prompted
func (e *SecretExpander) ExpandWithPrompt(name string, project *Project) (string, *failures.Failure) {
	if e.project == nil {
		e.project = project
	}
	if e.projectFile == nil {
		e.projectFile = project.Source()
	}

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", fail
	}

	value, fail := e.FetchSecret(name, e.isUser)
	if fail != nil && !fail.Type.Matches(secretsapi.FailUserSecretNotFound) {
		return "", fail
	}
	if fail == nil {
		return value, nil
	}

	def, fail := e.FetchDefinition(name, e.isUser)
	if fail != nil {
		return "", fail
	}

	scope := string(secretsapi.ScopeUser)
	if !e.isUser {
		scope = string(secretsapi.ScopeProject)
	}
	description := locale.T("secret_no_description")
	if def != nil && def.Description != "" {
		description = def.Description
	}

	print.Line(locale.Tr("secret_value_prompt_summary", name, description, scope, locale.T("secret_prompt_"+scope)))
	if value, fail = Prompter.InputSecret(locale.Tr("secret_value_prompt", name)); fail != nil {
		return "", FailInputSecretValue.New("secrets_err_value_prompt")
	}

	pj, fail := e.Project()
	if fail != nil {
		return "", fail
	}
	org, fail := e.Organization()
	if fail != nil {
		return "", fail
	}

	fail = secrets.Save(e.secretsClient, keypair, org, pj, e.isUser, name, value)

	if fail != nil {
		return "", fail
	}

	// Cache it so we're not repeatedly prompting for the same secret
	e.cachedSecrets[name] = value

	return value, nil
}
