package projects

import (
	"fmt"
	"strings"

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
	Name           string   `json:"name"`
	Organization   string   `json:"organization"`
	LocalCheckouts []string `json:"local_checkouts,omitempty" opts:"emptyNil"`
}

type configGetter interface {
	GetStringMapStringSlice(key string) map[string][]string
}

type Projects struct {
	out    output.Outputer
	auth   *authentication.Auth
	config configGetter
}

func NewProjects(outputer output.Outputer, auth *authentication.Auth, config configGetter) *Projects {
	return &Projects{outputer, auth, config}
}

func (r *Projects) Run() *failures.Failure {
	projectfile.CleanProjectMapping()

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
	platfomProjects := []projectWithOrg{}
	localProjects := []projectWithOrg{}
	localConfigProjects := r.config.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)
	for _, org := range orgs.Payload {
		orgProjects, err := model.FetchOrganizationProjects(org.URLname)
		if err != nil {
			return nil, err
		}
		for _, project := range orgProjects {
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
			if localPaths, ok := localConfigProjects[fmt.Sprintf("%s/%s", strings.ToLower(org.URLname), strings.ToLower(project.Name))]; ok {
				p.LocalCheckouts = localPaths
				localProjects = append(localProjects, p)
				continue
			}
			platfomProjects = append(platfomProjects, p)
		}
	}
	return append(localProjects, platfomProjects...), nil
}

func nameAndDescription(name, description string) string {
	limit := 32
	if len(description) > limit {
		description = fmt.Sprintf("%s...", description[0:limit])
	}

	return fmt.Sprintf("%s (%s)", name, description)
}
