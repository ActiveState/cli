package e2e

import (
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func cleanUser(t *testing.T, username string, auth *authentication.Auth) error {
	projects, err := getProjects(username, auth)
	if err != nil {
		return err
	}
	for _, proj := range projects {
		err = model.DeleteProject(username, proj.Name, auth)
		if err != nil {
			return err
		}
	}

	return deleteUser(username, auth)
}

func getProjects(org string, auth *authentication.Auth) ([]*mono_models.Project, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	params := projects.NewListProjectsParams()
	params.SetOrganizationName(org)
	listProjectsOK, err := authClient.Projects.ListProjects(params, auth.ClientAuth())
	if err != nil {
		return nil, err
	}

	return listProjectsOK.Payload, nil
}

func deleteUser(name string, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	params := users.NewDeleteUserParams()
	params.SetUsername(name)

	_, err = authClient.Users.DeleteUser(params, auth.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}
