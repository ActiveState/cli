package projdb

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/gql"
	"github.com/go-openapi/strfmt"
)

func NewProjectsRespDefaultMock(orgData TextToID) *gql.ProjectsResp {
	psr := &gql.ProjectsResp{
		Projects: []*gql.Project{
			&gql.Project{
				Description:    PtrToString("the CodeIntel project of ActiveState"),
				Name:           "CodeIntel",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 1)},
				CreatedBy:      NewStrfmtUUID(1, 1),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 12)},
				OrganizationID: orgData.ID("ActiveState"),
			},
			&gql.Project{
				Description:    PtrToString("the SecretProject project of SecretOrg"),
				Name:           "SecretProject",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 1)},
				CreatedBy:      NewStrfmtUUID(2, 2),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 12)},
				OrganizationID: orgData.ID("SecretOrg"),
			},
			&gql.Project{
				Description:    PtrToString("the example-proj of example-org"),
				Name:           "example-proj",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 10)},
				CreatedBy:      NewStrfmtUUID(3, 3),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 24 * 9)},
				OrganizationID: orgData.ID("example-org"),
			},
			&gql.Project{
				Description:    PtrToString("the sample-proj of example-org"),
				Name:           "sample-proj",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(3, 3),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("example-org"),
			},
			&gql.Project{
				Branches:       MakeBranchesBareMock(5, MakeStrfmtUUID(5, 5)),
				Description:    PtrToString("the example-proj of sample-org"),
				Name:           "example-proj",
				Added:          gql.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(4, 4),
				Changed:        gql.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("sample-org"),
			},
		},
	}

	for i, proj := range psr.Projects {
		n := i + 1
		projID := MakeStrfmtUUID(n, n)

		proj.ProjectID = projID

		if proj.Branches == nil {
			proj.Branches = MakeBranchesMock(n, 2, 2, projID)
		}
	}

	return psr
}

func MakeOrgDataDefaultMock() TextToID {
	return map[string]strfmt.UUID{
		"ActiveState": MakeStrfmtUUID(1, 1),
		"SecretOrg":   MakeStrfmtUUID(2, 2),
		"example-org": MakeStrfmtUUID(3, 3),
		"sample-org":  MakeStrfmtUUID(4, 4),
	}
}

type TextToID map[string]strfmt.UUID

func (m TextToID) ID(text string) strfmt.UUID {
	if id, ok := m[text]; ok {
		return id
	}
	panic(fmt.Sprintf("cannot find id by text %q", text))
}

func MakeStrfmtUUID(n, count int) strfmt.UUID {
	if n > 9999 || count > 9999 {
		panic("cannot make more than 9999 unique values")
	}
	return strfmt.UUID(
		fmt.Sprintf("%04d%04d-%04d-%04d-%04d-%04d%04d%04d", n, n, n, n, n, n, n, count),
	)
}

func NewStrfmtUUID(n, count int) *strfmt.UUID {
	id := MakeStrfmtUUID(n, count)
	return &id
}

func MakeBranchesMock(n, qty, main int, projID strfmt.UUID) gql.Branches {
	isMain := true

	var bs []*gql.Branch

	for i := 1; i <= qty; i++ {
		b := &gql.Branch{
			BranchID:  MakeStrfmtUUID(n, i),
			CommitID:  NewStrfmtUUID(n, i),
			ProjectID: &projID,
		}
		if i == main {
			b.Main = &isMain
		}
		bs = append(bs, b)
	}

	return bs
}

func MakeBranchesBareMock(n int, projID strfmt.UUID) gql.Branches {
	return MakeBranchesMock(n, 1, 1, projID)
}

func PtrToString(s string) *string {
	return &s
}
