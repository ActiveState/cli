package projdb

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/gql"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/go-openapi/strfmt"
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

func NewProjectsRespMock(orgData TextToID) *gql.ProjectsResp {
	return &gql.ProjectsResp{
		Projects: []*gql.Project{
			&gql.Project{
				Branches:       MakeBranchesMock(0, MakeStrfmtUUID(1)),
				Description:    PtrToString("the example-proj of example-org"),
				Name:           "example-proj",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 10)},
				CreatedBy:      NewStrfmtUUID(1),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 24 * 9)},
				OrganizationID: orgData.ID("example-org"),
				ProjectID:      MakeStrfmtUUID(1),
			},
			&gql.Project{
				Branches:       MakeBranchesMock(4, MakeStrfmtUUID(2)),
				Description:    PtrToString("the sample-proj of example-org"),
				Name:           "sample-proj",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(1),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("example-org"),
				ProjectID:      MakeStrfmtUUID(2),
			},
			&gql.Project{
				Branches:       MakeBranchesBareMock(8, MakeStrfmtUUID(3)),
				Description:    PtrToString("the example-proj of sample-org"),
				Name:           "example-proj",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(2),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("sample-org"),
				ProjectID:      MakeStrfmtUUID(3),
			},
			&gql.Project{
				Branches:       MakeBranchesMock(12, MakeStrfmtUUID(4)),
				Description:    PtrToString("the CodeIntel project of ActiveState"),
				Name:           "CodeIntel",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 1)},
				CreatedBy:      NewStrfmtUUID(3),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 12)},
				OrganizationID: orgData.ID("ActiveState"),
				ProjectID:      MakeStrfmtUUID(4),
			},
		},
	}
}

func MakeOrgDataMock() TextToID {
	return map[string]strfmt.UUID{
		"example-org": MakeStrfmtUUID(1),
		"sample-org":  MakeStrfmtUUID(2),
		"ActiveState": MakeStrfmtUUID(3),
	}
}

type TextToID map[string]strfmt.UUID

func (m TextToID) ID(text string) strfmt.UUID {
	if id, ok := m[text]; ok {
		return id
	}
	panic(fmt.Sprintf("cannot find id by text %q", text))
}

func MakeStrfmtUUID(n uint8) strfmt.UUID {
	return strfmt.UUID(
		fmt.Sprintf("%04d%04d-%04d-%04d-%04d-%04d%04d%04d", n, n, n, n, n, n, n, n),
	)
}

func NewStrfmtUUID(n uint8) *strfmt.UUID {
	id := MakeStrfmtUUID(n)
	return &id
}

func MakeBranchesMock(offset uint8, projID strfmt.UUID) gql.Branches {
	isMain := true

	return []*gql.Branch{
		&gql.Branch{
			BranchID:  MakeStrfmtUUID(offset + 1),
			CommitID:  NewStrfmtUUID(offset + 1),
			ProjectID: &projID,
		},
		&gql.Branch{
			BranchID:  MakeStrfmtUUID(offset + 2),
			CommitID:  NewStrfmtUUID(offset + 2),
			Main:      &isMain,
			ProjectID: &projID,
		},
	}
}

func MakeBranchesBareMock(offset uint8, projID strfmt.UUID) gql.Branches {
	isMain := true

	return []*gql.Branch{
		&gql.Branch{
			BranchID:  MakeStrfmtUUID(offset + 1),
			Main:      &isMain,
			ProjectID: &projID,
		},
	}
}

func PtrToString(s string) *string {
	return &s
}
