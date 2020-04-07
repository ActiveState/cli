package integration

import (
	"log"
	"os"

	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func cleanUser(username string) error {
	err := os.Setenv("ACTIVESTATE_API_HOST", "platform.activestate.com")
	if err != nil {
		return err
	}
	defer func() {
		os.Setenv("ACTIVESTATE_API_HOST", "platform.testing.tld")
	}()

	fail := auth.AuthenticateWithCredentials(&mono_models.Credentials{
		Token: os.Getenv("PLATFORM_API_TOKEN"),
	})
	if fail != nil {
		log.Fatalf("Could not authenticate test cleaning user: %v", fail)
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
	listProjectsOK, err := authentication.Get().Client().Projects.ListProjects(params, authentication.ClientAuth())
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
