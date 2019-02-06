package expander

import (
	"strings"

	"github.com/ActiveState/cli/internal/secrets"

	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/pkg/projectfile"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// FailExpandNoProjectDefined is used when error arises from an expander function being called without a project
var FailExpandNoProjectDefined = failures.Type("expander.fail.secrets.expand.noproject")

// FailInputSecretValue is used when error arises from user providing a secret value.
var FailInputSecretValue = failures.Type("expander.fail.secrets.input.value", failures.FailUserInput)

// SecretExpander takes care of expanding variables that we know to be secrets
type SecretExpander struct {
	secretsClient *secretsapi.Client
	keypair       keypairs.Keypair
	organization  *models.Organization
	project       *models.Project
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
func (e *SecretExpander) Organization() (*models.Organization, *failures.Failure) {
	if e.projectFile == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}
	var fail *failures.Failure
	if e.organization == nil {
		e.organization, fail = organizations.FetchByURLName(e.projectFile.Owner)
		if fail != nil {
			return nil, fail
		}
	}

	return e.organization, nil
}

// Project acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Project() (*models.Project, *failures.Failure) {
	if e.projectFile == nil {
		return nil, FailExpandNoProjectDefined.New(locale.T("secrets_err_expand_noproject"))
	}
	var fail *failures.Failure
	if e.project == nil {
		e.project, fail = projects.FetchByName(e.projectFile.Owner, e.projectFile.Name)
		if fail != nil {
			return nil, fail
		}
	}

	return e.project, nil
}

// Secrets acts as a caching layer, and ensures that we have a projectfile
func (e *SecretExpander) Secrets() ([]*secretsModels.UserSecret, *failures.Failure) {
	org, fail := e.Organization()
	if fail != nil {
		return nil, fail
	}
	if e.secrets == nil {
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

	userSecret, fail := e.FindSecretWithHighestPriority(variable)
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

// FindSecretWithHighestPriority will find the most appropriately scoped secret from the provided collection given
// the provided SecretSpec. This function would like to find a secret with the following priority:
//
// 0. name match, case-insensitive (obvious given)
// 1. secret is user+project-scoped and project matches current project
// 2. secret is user-scoped
// 3. secret is project-scoped and spec does not require user-scope only
// 4. secret is org-scoped and spec does not require user and/or project-scope only
//
// Thus, if secrets are found matching priority 1 and 3, the priority 1 secret is returned. If no secret
// is found, nil is returned.
func (e *SecretExpander) FindSecretWithHighestPriority(variable *projectfile.Variable) (*secretsModels.UserSecret, *failures.Failure) {
	secrets, fail := e.Secrets()
	if fail != nil {
		return nil, fail
	}

	project, fail := e.Project()
	if fail != nil {
		return nil, fail
	}

	projectIDStr := project.ProjectID.String()

	var selectedSecret *secretsModels.UserSecret
	for _, userSecret := range secrets {
		secretProjectIDStr := userSecret.ProjectID.String()
		secretRequiresUser := *userSecret.IsUser
		secretRequiresProject := secretProjectIDStr != ""

		if !strings.EqualFold(*userSecret.Name, variable.Name) {
			continue
		} else if secretRequiresUser && secretProjectIDStr == projectIDStr {
			// priority 1 match
			return userSecret, nil
		} else if variable.Value.Share == nil && !secretRequiresUser {
			// user scoped secret required (priority 2 failure)
			continue
		} else if secretRequiresProject && secretProjectIDStr != projectIDStr {
			// this is a project secret but project id's don't match (priority 3 failure)
			continue
		} else if variable.Value.PullFrom != nil && *variable.Value.PullFrom == projectfile.VariablePullFromProject && !secretRequiresUser && !secretRequiresProject {
			// org scoped secret when project or user scope required (priority 4 failure)
			continue
		}

		if selectedSecret == nil {
			// basic requirements met and nothing else selected yet
			selectedSecret = userSecret
			continue
		} else if secretRequiresUser && !*selectedSecret.IsUser {
			// priority 2 match
			selectedSecret = userSecret
			continue
		} else if secretProjectIDStr == projectIDStr {
			// priority 3 match
			selectedSecret = userSecret
			continue
		}
	}

	return selectedSecret, nil
}

// SecretExpanderFunc defines what our expander functions will be returning
type SecretExpanderFunc func(variable *projectfile.Variable, projectFile *projectfile.Project) (string, *failures.Failure)

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

	userSecret, fail := e.FindSecretWithHighestPriority(variable)
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
		if value, fail = promptForValue(variable); fail != nil {
			return "", fail
		}

		project, fail := e.Project()
		if fail != nil {
			return "", fail
		}
		org, fail := e.Organization()
		if fail != nil {
			return "", fail
		}

		if variable.Value.PullFrom != nil && *variable.Value.PullFrom == projectfile.VariablePullFromProject {
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

func promptForValue(variable *projectfile.Variable) (string, *failures.Failure) {
	var value string
	// TODO: remove scope prop from locale.Tr
	var prompt = &survey.Password{Message: locale.Tr("secret_value_prompt", "SCOPE", variable.Name)}
	if err := survey.AskOne(prompt, &value, nil); err != nil {
		return "", FailInputSecretValue.New("variables_err_value_prompt")
	}
	return value, nil
}
