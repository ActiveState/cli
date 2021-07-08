package e2e

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func cleanUser(t *testing.T, username string) error {
	if os.Getenv(constants.APIHostEnvVarName) == "" {
		err := os.Setenv(constants.APIHostEnvVarName, constants.DefaultAPIHost)
		if err != nil {
			return err
		}
		defer func() {
			os.Unsetenv(constants.APIHostEnvVarName)
		}()
	}

	err := auth.AuthenticateWithCredentials(&mono_models.Credentials{
		Token: os.Getenv("PLATFORM_API_TOKEN"),
	})
	if err != nil {
		return err
	}

	projects, err := getProjects(username)
	if err != nil {
		return err
	}
	for _, proj := range projects {
		err = deleteProject(username, proj.Name)
		if err != nil {
			return err
		}
	}

	return deleteUser(username)
}

func getProjects(org string) ([]*mono_models.Project, error) {
	params := projects.NewListProjectsParams()
	params.SetOrganizationName(org)
	listProjectsOK, err := authentication.LegacyGet().Client().Projects.ListProjects(params, authentication.ClientAuth())
	if err != nil {
		return nil, err
	}

	return listProjectsOK.Payload, nil
}

func deleteProject(org, name string) error {
	params := projects.NewDeleteProjectParams()
	params.SetOrganizationName(org)
	params.SetProjectName(name)

	_, err := authentication.Client().Projects.DeleteProject(params, authentication.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}

func deleteUser(name string) error {
	params := users.NewDeleteUserParams()
	params.SetUsername(name)

	_, err := authentication.Client().Users.DeleteUser(params, authentication.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}

