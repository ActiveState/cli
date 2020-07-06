package projects

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Holds a union of project and organization parameters.
type projectWithOrg struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Organization string `json:"organization"`
}

type Projects struct {
	out  output.Outputer
	auth *authentication.Auth
}

func NewProjects(outputer output.Outputer, auth *authentication.Auth) *Projects {
	return &Projects{outputer, auth}
}

func (r *Projects) Run() *failures.Failure {
	projectsList, fail := r.fetchProjects()
	if fail != nil {
		return fail.WithDescription(locale.T("project_err"))
	}

	if len(projectsList) == 0 {
		r.out.Print(locale.T("project_empty"))
		return nil
	}

	r.out.Print(projectsList)
	return nil
}

func (r *Projects) fetchProjects() ([]projectWithOrg, *failures.Failure) {
	orgParams := organizations.NewListOrganizationsParams()
	memberOnly := true
	orgParams.SetMemberOnly(&memberOnly)
	orgs, err := r.auth.Client().Organizations.ListOrganizations(orgParams, authentication.ClientAuth())
	if err != nil {
		if api.ErrorCode(err) == 401 {
			return nil, api.FailAuth.New("err_api_not_authenticated")
		}
		return nil, api.FailUnknown.Wrap(err)
	}
	projectsList := []projectWithOrg{}
	for _, org := range orgs.Payload {
		orgProjects, err := model.FetchOrganizationProjects(org.URLname)
		if err != nil {
			return nil, err
		}
		for _, project := range orgProjects {
			desc := ""
			if project.Description != nil {
				desc = *project.Description
			}
			projectsList = append(projectsList, projectWithOrg{project.Name, desc, org.Name})
		}
	}
	projectfile.CleanStaleConfig()
	return projectsList, nil
}
