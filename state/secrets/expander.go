package secrets

import (
	"strings"

	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	// FailUnrecognizedSecretSpec is used when no handler func is found for an Expander.
	FailUnrecognizedSecretSpec = failures.Type("secrets.fail.unrecognized.secret_spec", variables.FailExpandVariable)
)

// NewExpander creates an ExpanderFunc which can decrypt stored user secrets.
func NewExpander(secretsClient *secretsapi.Client) variables.ExpanderFunc {
	return func(name string, projectFile *projectfile.Project) (string, *failures.Failure) {
		spec := projectFile.Secrets.GetByName(name)
		if spec == nil {
			return "", FailUnrecognizedSecretSpec.New("secrets_expand_err_spec_undefined", name)
		}

		org, failure := organizations.FetchByURLName(projectFile.Owner)
		if failure != nil {
			return "", failure
		}

		proj, failure := projects.FetchByName(org.Urlname, projectFile.Name)
		if failure != nil {
			return "", failure
		}

		kp, failure := keypairs.Fetch(secretsClient)
		if failure != nil {
			return "", failure
		}

		userSecrets, failure := fetchAll(secretsClient, org)
		if failure != nil {
			return "", failure
		}

		userSecret := findSecretWithHighestPriority(userSecrets, spec, proj)
		if userSecret == nil {
			return "", secretsapi.FailUserSecretNotFound.New("secrets_expand_err_not_found", name)
		}

		decrBytes, failure := kp.DecodeAndDecrypt(*userSecret.Value)
		if failure != nil {
			return "", failure
		}

		return string(decrBytes), nil
	}
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
func findSecretWithHighestPriority(userSecrets []*secretsModels.UserSecret, spec *projectfile.SecretSpec, project *models.Project) *secretsModels.UserSecret {
	if project == nil {
		return nil
	}

	projectIDStr := project.ProjectID.String()

	var selectedSecret *secretsModels.UserSecret
	for _, userSecret := range userSecrets {
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
