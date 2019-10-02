package projdb

import (
	"time"

	"github.com/ActiveState/cli/internal/gql"
	"github.com/ActiveState/cli/internal/gqlclient"
)

func NewProjectClient(isTest bool, endpoint string, hdr gqlclient.Header) (gql.ProjectClient, error) {
	switch isTest {
	case true:
		return NewMock(), nil
	default:
		timeout := time.Second * 16
		return New(endpoint, hdr, timeout)
	}
}

type ProjDB struct {
	gc *gqlclient.GQLClient
}

func New(endpoint string, hdr gqlclient.Header, timeout time.Duration) (*ProjDB, error) {
	db := ProjDB{
		gc: gqlclient.New(endpoint, hdr, timeout),
	}

	return &db, nil
}

type Mock struct {
	ProjectsResp *gql.ProjectsResp
}

func NewMock() *Mock {
	return &Mock{
		ProjectsResp: &gql.ProjectsResp{
			Projects: []*gql.Project{&gql.Project{}},
		},
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

	return resp.FirstToProjectResp()
}

func (mk *Mock) ProjectByOrgAndName(org, name string) (*gql.ProjectResp, error) {
	return mk.ProjectsResp.FirstToProjectResp()
}
