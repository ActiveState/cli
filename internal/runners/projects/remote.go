package projects

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func (r *Projects) RunRemote(params *Params) *failures.Failure {
	projectfile.CleanProjectMapping()

	projectsList, fail := r.fetchProjects(params.Local)
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

func (r *Projects) fetchProjects(onlyLocal bool) (projectWithOrgs, *failures.Failure) {
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
	var projects projectWithOrgs = []projectWithOrg{}
	localConfigProjects := r.config.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	for _, org := range orgs.Payload {
		platformOrgProjects, err := model.FetchOrganizationProjects(org.URLname)
		if err != nil {
			return nil, err
		}

		orgProjects := make([]projectWithOrg, 0, len(platformOrgProjects))
		for _, project := range platformOrgProjects {
			p := projectWithOrg{
				Name:         project.Name,
				Organization: org.Name,
			}

			// Description can be non-nil but also empty
			if project.Description != nil && *project.Description != "" {
				p.Name = nameAndDescription(project.Name, *project.Description)
			}

			// Viper lowers all map keys so we must do the same here
			// in order to retrieve the locally cached projects
			localPaths, ok := localConfigProjects[fmt.Sprintf("%s/%s", strings.ToLower(org.URLname), strings.ToLower(project.Name))]
			if ok {
				p.LocalCheckouts = localPaths
			} else if onlyLocal {
				continue
			}
			orgProjects = append(orgProjects, p)
		}

		projects = append(projects, orgProjects...)
	}
	sort.SliceStable(projects, func(i, j int) bool {
		return (projects[i].LocalCheckouts != nil && projects[j].LocalCheckouts == nil) ||
			projects[i].Organization < projects[j].Organization
	})

	return projects, nil
}

func nameAndDescription(name, description string) string {
	limit := 32
	if len(description) > limit {
		description = fmt.Sprintf("%s...", description[0:limit])
	}

	return fmt.Sprintf("%s (%s)", name, description)
}
