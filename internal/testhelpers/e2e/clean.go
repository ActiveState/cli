package e2e

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func cleanUser(t *testing.T, username string, auth *authentication.Auth) error {
	err := authenticate(auth)
	if err != nil {
		return err
	}

	projects, err := getProjects(username, auth)
	if err != nil {
		return err
	}
	for _, proj := range projects {
		err = deleteProject(username, proj.Name, auth)
		if err != nil {
			return err
		}
	}

	return deleteUser(username, auth)
}

func getProjects(org string, auth *authentication.Auth) ([]*mono_models.Project, error) {
	params := projects.NewListProjectsParams()
	params.SetOrganizationName(org)
	listProjectsOK, err := auth.Client().Projects.ListProjects(params, auth.ClientAuth())
	if err != nil {
		return nil, err
	}

	return listProjectsOK.Payload, nil
}

func deleteProject(org, name string, auth *authentication.Auth) error {
	params := projects.NewDeleteProjectParams()
	params.SetOrganizationName(org)
	params.SetProjectName(name)

	_, err := auth.Client().Projects.DeleteProject(params, auth.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}

func deleteUser(name string, auth *authentication.Auth) error {
	params := users.NewDeleteUserParams()
	params.SetUsername(name)

	_, err := auth.Client().Users.DeleteUser(params, auth.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}
