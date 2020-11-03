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

	fail := auth.AuthenticateWithCredentials(&mono_models.Credentials{
		Token: os.Getenv("PLATFORM_API_TOKEN"),
	})
	if fail != nil {
		return fail.ToError()
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

	client, err := authentication.Get().Client()
	if err != nil {
		return nil, err
	}

	listProjectsOK, err := client.Projects.ListProjects(params, authentication.ClientAuth())
	if err != nil {
		return nil, err
	}

	return listProjectsOK.Payload, nil
}

func deleteProject(org, name string) error {
	params := projects.NewDeleteProjectParams()
	params.SetOrganizationName(org)
	params.SetProjectName(name)

	client, err := authentication.Get().Client()
	if err != nil {
		return err
	}

	if _, err = client.Projects.DeleteProject(params, authentication.ClientAuth()); err != nil {
		return err
	}

	return nil
}

func deleteUser(name string) error {
	params := users.NewDeleteUserParams()
	params.SetUsername(name)

	client, err := authentication.Get().Client()
	if err != nil {
		return err
	}

	if _, err = client.Users.DeleteUser(params, authentication.ClientAuth()); err != nil {
		return err
	}

	return nil
}
