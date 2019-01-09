package secrets

import (
	"strings"

	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

var (
	// FailUnrecognizedSecretSpec is used when no handler func is found for an Expander.
	FailUnrecognizedSecretSpec = failures.Type("secrets.fail.unrecognized.secret_spec", variables.FailExpandVariable)

	// FailInputSecretValue is used when error arises from user providing a secret value.
	FailInputSecretValue = failures.Type("secrets.fail.input.value", failures.FailUserInput)
)

// NewExpander creates an ExpanderFunc which can retrieve and decrypt stored user secrets.
func NewExpander(secretsClient *secretsapi.Client) variables.ExpanderFunc {
	return newContextMemoizingExpanderFunc(secretsClient, expandSecret)
}

// NewPromptingExpander creates an ExpanderFunc which can retrieve and decrypt stored user secrets. Additionally,
// it will prompt the user to provide a value for a secret -- in the event none is found -- and save the new
// value with the secrets service.
func NewPromptingExpander(secretsClient *secretsapi.Client) variables.ExpanderFunc {
	return newContextMemoizingExpanderFunc(secretsClient, func(expanderCtx *expanderContext, spec *projectfile.SecretSpec) (string, *failures.Failure) {
		value, failure := expandSecret(expanderCtx, spec)
		if failure != nil && failure.Type.Matches(secretsapi.FailUserSecretNotFound) {
			if value, failure = promptForValue(); failure != nil {
				return "", failure
			}

			if spec.IsProject {
				failure = saveUserSecret(secretsClient, expanderCtx.Keypair, expanderCtx.Organization, expanderCtx.Project, spec.IsUser, spec.Name, value)
			} else {
				failure = saveUserSecret(secretsClient, expanderCtx.Keypair, expanderCtx.Organization, nil, spec.IsUser, spec.Name, value)
			}

			if failure != nil {
				return "", failure
			}
		}

		return value, nil
	})
}

type expanderContext struct {
	Keypair       keypairs.Keypair
	Organization  *models.Organization
	Project       *models.Project
	UserSecrets   []*secretsModels.UserSecret
	cachedSecrets map[string]string
}

func buildExpanderContext(secretsClient *secretsapi.Client, projectFile *projectfile.Project) (*expanderContext, *failures.Failure) {
	kp, failure := loadKeypairFromConfigDir()
	if failure != nil {
		return nil, failure
	}

	org, failure := organizations.FetchByURLName(projectFile.Owner)
	if failure != nil {
		return nil, failure
	}

	proj, failure := projects.FetchByName(org.Urlname, projectFile.Name)
	if failure != nil {
		return nil, failure
	}

	userSecrets, failure := fetchAll(secretsClient, org)
	if failure != nil {
		return nil, failure
	}

	return &expanderContext{
		Keypair:       kp,
		Organization:  org,
		Project:       proj,
		UserSecrets:   userSecrets,
		cachedSecrets: map[string]string{},
	}, nil
}

type secretExpanderFunc func(expanderCtx *expanderContext, spec *projectfile.SecretSpec) (string, *failures.Failure)

func newContextMemoizingExpanderFunc(secretsClient *secretsapi.Client, fn secretExpanderFunc) variables.ExpanderFunc {
	// memoized context
	var expanderCtx *expanderContext

	return func(name string, projectFile *projectfile.Project) (string, *failures.Failure) {
		var failure *failures.Failure

		spec := projectFile.Secrets.GetByName(name)
		if spec == nil {
			return "", FailUnrecognizedSecretSpec.New("secrets_expand_err_spec_undefined", name)
		}

		if expanderCtx == nil {
			expanderCtx, failure = buildExpanderContext(secretsClient, projectFile)
			if failure != nil {
				return "", failure
			}
		}
		return fn(expanderCtx, spec)
	}
}

func expandSecret(expanderCtx *expanderContext, spec *projectfile.SecretSpec) (string, *failures.Failure) {
	if knownValue, exists := expanderCtx.cachedSecrets[spec.Name]; exists {
		return knownValue, nil
	}

	var failure *failures.Failure

	userSecret := findSecretWithHighestPriority(expanderCtx, spec)
	if userSecret == nil {
		return "", secretsapi.FailUserSecretNotFound.New("secrets_expand_err_not_found", spec.Name)
	}

	decrBytes, failure := expanderCtx.Keypair.DecodeAndDecrypt(*userSecret.Value)
	if failure != nil {
		return "", failure
	}

	secretValue := string(decrBytes)
	expanderCtx.cachedSecrets[spec.Name] = secretValue
	return secretValue, nil
}

// findSecretWithHighestPriority will find the most appropriately scoped secret from the provided collection given
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
func findSecretWithHighestPriority(expanderCtx *expanderContext, spec *projectfile.SecretSpec) *secretsModels.UserSecret {
	projectIDStr := expanderCtx.Project.ProjectID.String()

	var selectedSecret *secretsModels.UserSecret
	for _, userSecret := range expanderCtx.UserSecrets {
		secretProjectIDStr := userSecret.ProjectID.String()
		secretRequiresUser := *userSecret.IsUser
		secretRequiresProject := secretProjectIDStr != ""

		if !strings.EqualFold(*userSecret.Name, spec.Name) {
			continue
		} else if secretRequiresUser && secretProjectIDStr == projectIDStr {
			// priority 1 match
			return userSecret
		} else if spec.IsUser && !secretRequiresUser {
			// user scoped secret required (priority 2 failure)
			continue
		} else if secretRequiresProject && secretProjectIDStr != projectIDStr {
			// this is a project secret but project id's don't match (priority 3 failure)
			continue
		} else if spec.IsProject && !secretRequiresUser && !secretRequiresProject {
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
	return selectedSecret
}

func promptForValue() (string, *failures.Failure) {
	var value string
	var prompt = &survey.Password{Message: locale.T("secret_value_prompt")}
	if err := survey.AskOne(prompt, &value, nil); err != nil {
		return "", FailInputSecretValue.New("secrets_err_value_prompt")
	}
	return value, nil
}
