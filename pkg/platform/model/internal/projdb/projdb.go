package projdb

import (
	"time"

	"github.com/ActiveState/cli/internal/dbm"
	"github.com/ActiveState/cli/internal/gqlclient"
)

func NewProvider(isTest bool, endpoint string, hdr gqlclient.Header) (dbm.ProjectProvider, error) {
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
	ProjectsResp *dbm.ProjectsResp
}

func NewMock() *Mock {
	return &Mock{
		ProjectsResp: &dbm.ProjectsResp{
			Projects: []*dbm.Project{&dbm.Project{}},
		},
	}
}

func (db *ProjDB) ProjectByOrgAndName(org, name string) (*dbm.ProjectResp, error) {
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

	var resp dbm.ProjectResp
	err := db.gc.Run(req, &resp)
	return &resp, err
}

func (mk *Mock) ProjectByOrgAndName(org, name string) (*dbm.ProjectResp, error) {
	return &dbm.ProjectResp{mk.ProjectsResp.Projects[0]}, nil
}
