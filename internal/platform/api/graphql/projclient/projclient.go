package projclient

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/platform/api/client"
)

type ProjClient struct {
	gc *gqlclient.GQLClient
}

func New(gc *gqlclient.GQLClient) (*ProjClient, error) {
	pc := ProjClient{
		gc: gc,
	}

	return &pc, nil
}

type Mock struct {
	ProjectsResp *client.ProjectsResp
	OrgData      TextToID
}

func NewMock(pr *client.ProjectsResp, orgData TextToID) *Mock {
	return &Mock{
		ProjectsResp: pr,
		OrgData:      orgData,
	}
}

func (pc *ProjClient) ProjectByOrgAndName(org, name string) (*client.ProjectResp, error) {
	req := pc.gc.NewRequest(`
query ($org: String, $name: String) {
  projects(where: {name: {_eq: $name}, organization: {url_name: {_eq: $org}}}, limit: 1) {
    branches {
      branch_id
      commit_id
      main
      project_id
      tracking_type
      tracks
      label
    }
    description
    name
    added
    created_by
    forked_from
    forked_project {
      name
      organization {
        url_name
      }
    }
    changed
    managed
    organization_id
    private
    project_id
    repo_url
  }
}
`)
	req.Var("org", org)
	req.Var("name", name)

	var resp client.ProjectsResp
	if err := pc.gc.Run(req, &resp); err != nil {
		return nil, err
	}

	return resp.ProjectToProjectResp(0)
}

func (mk *Mock) ProjectByOrgAndName(org, name string) (*client.ProjectResp, error) {
	orgID, ok := mk.OrgData[org]
	if !ok {
		return mk.ProjectsResp.ProjectToProjectResp(-1)
	}

	for i, p := range mk.ProjectsResp.Projects {
		if p.Name == name && p.OrganizationID == orgID {
			return mk.ProjectsResp.ProjectToProjectResp(i)
		}
	}

	return mk.ProjectsResp.ProjectToProjectResp(-1)
}
