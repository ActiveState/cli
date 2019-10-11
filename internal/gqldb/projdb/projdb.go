package projdb

import (
	"github.com/ActiveState/cli/internal/gql"
	"github.com/ActiveState/cli/internal/gqlclient"
)

type ProjDB struct {
	gc *gqlclient.GQLClient
}

func New(gc *gqlclient.GQLClient) (*ProjDB, error) {
	db := ProjDB{
		gc: gc,
	}

	return &db, nil
}

type Mock struct {
	ProjectsResp *gql.ProjectsResp
	OrgData      TextToID
}

func NewMock(pr *gql.ProjectsResp, orgData TextToID) *Mock {
	return &Mock{
		ProjectsResp: pr,
		OrgData:      orgData,
	}
}

func (db *ProjDB) ProjectByOrgAndName(org, name string) (*gql.ProjectResp, error) {
	req := db.gc.NewRequest(`
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

	var resp gql.ProjectsResp
	if err := db.gc.Run(req, &resp); err != nil {
		return nil, err
	}

	return resp.ProjectToProjectResp(0)
}

func (mk *Mock) ProjectByOrgAndName(org, name string) (*gql.ProjectResp, error) {
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
