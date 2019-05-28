package expander

import (
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/secrets"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailExpandNoProjectDefined is used when error arises from an expander function being called without a project
var FailExpandNoProjectDefined = failures.Type("expander.fail.secrets.expand.noproject")

// FailInputSecretValue is used when error arises from user providing a secret value.
var FailInputSecretValue = failures.Type("expander.fail.secrets.input.value", failures.FailUserInput)

// SecretExpander takes care of expanding variables that we know to be secrets
type SecretExpander struct {
	secretsClient *secretsapi.Client
	keypair       keypairs.Keypair
	organization  *mono_models.Organization
	project       *mono_models.Project
	projectFile   *projectfile.Project
	secrets       []*secretsModels.UserSecret
	cachedSecrets map[string]string
}

// NewSecretExpander returns a new instance of SecretExpander
func NewSecretExpander(secretsClient *secretsapi.Client) *SecretExpander {
	return &SecretExpander{
		secretsClient: secretsClient,
		cachedSecrets: map[string]string{},
	}
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
	if e.projectFile == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}
	var fail *failures.Failure
	if e.organization == nil {
		e.organization, fail = model.FetchOrgByURLName(e.projectFile.Owner)
		if fail != nil {
			return nil, fail
		}
	}

	return e.organization, nil
}

// Project acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Project() (*mono_models.Project, *failures.Failure) {
	if e.projectFile == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}
	var fail *failures.Failure
	if e.project == nil {
		e.project, fail = model.FetchProjectByName(e.projectFile.Owner, e.projectFile.Name)
		if fail != nil {
			return nil, fail
		}
	}

	return e.project, nil
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

// FetchSecret retrieves the secret associated with a variable
func (e *SecretExpander) FetchSecret(variable *projectfile.Variable) (string, *failures.Failure) {
	if knownValue, exists := e.cachedSecrets[variable.Name]; exists {
		return knownValue, nil
	}

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", nil
	}

	userSecret, fail := e.FindSecret(variable)
	if fail != nil {
		return "", fail
	}
	if userSecret == nil {
		return "", secretsapi.FailUserSecretNotFound.New("secrets_expand_err_not_found", variable.Name)
	}

	decrBytes, fail := keypair.DecodeAndDecrypt(*userSecret.Value)
	if fail != nil {
		return "", fail
	}

	e.cachedSecrets[variable.Name] = string(decrBytes)
	return e.cachedSecrets[variable.Name], nil
}

// FindSecret will find the secret appropriate for the current project
func (e *SecretExpander) FindSecret(variable *projectfile.Variable) (*secretsModels.UserSecret, *failures.Failure) {
	secrets, fail := e.Secrets()
	if fail != nil {
		return nil, fail
	}

	project, fail := e.Project()
	if fail != nil {
		return nil, fail
	}

	projectID := project.ProjectID.String()
	variableRequiresUser := variable.Value.Share == nil
	variableRequiresProject := variable.Value.Store == nil || *variable.Value.Store == projectfile.VariableStoreProject

	for _, userSecret := range secrets {
		secretProjectID := userSecret.ProjectID.String()
		secretRequiresUser := userSecret.IsUser != nil && *userSecret.IsUser
		secretRequiresProject := secretProjectID != ""

		nameMatches := strings.EqualFold(*userSecret.Name, variable.Name)
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
type SecretFunc func(variable *projectfile.Variable, projectFile *projectfile.Project) (string, *failures.Failure)

// Expand will expand a variable to a secret value, if no secret exists it will return an empty string
func (e *SecretExpander) Expand(variable *projectfile.Variable, projectFile *projectfile.Project) (string, *failures.Failure) {
	if e.projectFile == nil {
		e.projectFile = projectFile
	}

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", fail
	}

	if knownValue, exists := e.cachedSecrets[variable.Name]; exists {
		return knownValue, nil
	}

	userSecret, fail := e.FindSecret(variable)
	if fail != nil {
		return "", fail
	}
	if userSecret == nil {
		return "", secretsapi.FailUserSecretNotFound.New("variables_expand_err_not_found", variable.Name)
	}

	decrBytes, fail := keypair.DecodeAndDecrypt(*userSecret.Value)
	if fail != nil {
		return "", fail
	}

	secretValue := string(decrBytes)
	e.cachedSecrets[variable.Name] = secretValue
	return secretValue, nil
}

// ExpandWithPrompt will expand a variable to a secret value, if no secret exists the user will be prompted
func (e *SecretExpander) ExpandWithPrompt(variable *projectfile.Variable, projectFile *projectfile.Project) (string, *failures.Failure) {
	if e.projectFile == nil {
		e.projectFile = projectFile
	}

	keypair, fail := e.KeyPair()
	if fail != nil {
		return "", fail
	}

	value, fail := e.FetchSecret(variable)
	if fail != nil && fail.Type.Matches(secretsapi.FailUserSecretNotFound) {
		// TODO: remove scope prop from locale.Tr
		if value, fail = Prompter.InputSecret(locale.Tr("secret_value_prompt", "SCOPE", variable.Name)); fail != nil {
			return "", FailInputSecretValue.New("variables_err_value_prompt")
		}

		project, fail := e.Project()
		if fail != nil {
			return "", fail
		}
		org, fail := e.Organization()
		if fail != nil {
			return "", fail
		}

		if variable.Value.Store != nil && *variable.Value.Store == projectfile.VariableStoreProject {
			fail = secrets.Save(e.secretsClient, keypair, org, project, variable.Value.Share == nil, variable.Name, value)
		} else {
			fail = secrets.Save(e.secretsClient, keypair, org, nil, variable.Value.Share == nil, variable.Name, value)
		}

		if fail != nil {
			return "", fail
		}
	}

	return value, nil
}
