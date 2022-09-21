package e2e

import (
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/users"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

func (s *Session) cleanUser(username string) error {
	projects, err := s.getProjects(username)
	if err != nil {
		return err
	}
	for _, proj := range projects {
		err = s.DeleteProject(username, proj.Name)
		if err != nil {
			return err
		}
	}

	return s.deleteUser(username)
}

func (s *Session) getProjects(org string) ([]*mono_models.Project, error) {
	params := projects.NewListProjectsParams()
	params.SetOrganizationName(org)
	listProjectsOK, err := s.auth.Client().Projects.ListProjects(params, s.auth.ClientAuth())
	if err != nil {
		return nil, err
	}

	return listProjectsOK.Payload, nil
}

func (s *Session) DeleteProject(org, name string) error {
	if s.auth == nil {
		return nil // cannot do anything
	}

	params := projects.NewDeleteProjectParams()
	params.SetOrganizationName(org)
	params.SetProjectName(name)

	_, err := s.auth.Client().Projects.DeleteProject(params, s.auth.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}

func (s *Session) deleteUser(name string) error {
	params := users.NewDeleteUserParams()
	params.SetUsername(name)

	_, err := s.auth.Client().Users.DeleteUser(params, s.auth.ClientAuth())
	if err != nil {
		return err
	}

	return nil
}
