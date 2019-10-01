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
query {
  projects(where: {name: {_eq: "xxample"}, organization: {url_name: {_eq: "davedx"}}}, limit: 1) {
    organization {
      deleted
      display_name
      url_name
    }
    branches {
      commit_id
      main
      project_id
      tracking_type
      tracks
    }
    description
    languages
    name
  }
}
`)

	var resp gql.ProjectResp
	err := db.gc.Run(req, &resp)
	return &resp, err
}

func (mk *Mock) ProjectByOrgAndName(org, name string) (*gql.ProjectResp, error) {
	return &gql.ProjectResp{Project: mk.ProjectsResp.Projects[0]}, nil
}
