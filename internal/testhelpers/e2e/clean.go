package e2e

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

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
