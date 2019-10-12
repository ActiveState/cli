package projclient

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/platform/api/client"
	"github.com/ActiveState/cli/internal/platform/api/model"
	"github.com/go-openapi/strfmt"
)

func NewProjectsRespDefaultMock(orgData TextToID) *client.ProjectsResp {
	psr := &client.ProjectsResp{
		Projects: []*model.Project{
			&model.Project{
				Branches:       MakeBranchesBareMock(1, MakeStrfmtUUID(1, 1)),
				Description:    PtrToString("the CodeIntel project of ActiveState"),
				Name:           "CodeIntel",
				Added:          model.Time{Time: time.Now().Add(-time.Hour * 24 * 1)},
				CreatedBy:      NewStrfmtUUID(1, 1),
				Changed:        model.Time{Time: time.Now().Add(-time.Hour * 12)},
				OrganizationID: orgData.ID("ActiveState"),
			},
			&model.Project{
				Branches:       MakeBranchesBareMock(2, MakeStrfmtUUID(2, 2)),
				Description:    PtrToString("the SecretProject project of SecretOrg"),
				Name:           "SecretProject",
				Added:          model.Time{Time: time.Now().Add(-time.Hour * 24 * 1)},
				CreatedBy:      NewStrfmtUUID(2, 2),
				Changed:        model.Time{Time: time.Now().Add(-time.Hour * 12)},
				OrganizationID: orgData.ID("SecretOrg"),
			},
			&model.Project{
				Branches:       MakeBranchesBareMock(3, MakeStrfmtUUID(3, 3)),
				Description:    PtrToString("the example-proj of example-org"),
				Name:           "example-proj",
				Added:          model.Time{Time: time.Now().Add(-time.Hour * 24 * 10)},
				CreatedBy:      NewStrfmtUUID(3, 3),
				Changed:        model.Time{Time: time.Now().Add(-time.Hour * 24 * 9)},
				OrganizationID: orgData.ID("example-org"),
			},
			&model.Project{
				Description:    PtrToString("the sample-proj of example-org"),
				Name:           "sample-proj",
				Added:          model.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(3, 3),
				Changed:        model.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("example-org"),
			},
			&model.Project{
				Branches:       MakeBranchesBareMock(5, MakeStrfmtUUID(5, 5)),
				Description:    PtrToString("the example-proj of sample-org"),
				Name:           "example-proj",
				Added:          model.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(4, 4),
				Changed:        model.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("sample-org"),
			},
			&model.Project{
				Branches:       MakeBranchesMock(1, 1, 1, MakeStrfmtUUID(6, 6)),
				Description:    PtrToString("the string of string"),
				Name:           "string",
				Added:          model.Time{Time: time.Now().Add(-time.Hour * 24 * 3)},
				CreatedBy:      NewStrfmtUUID(4, 4),
				Changed:        model.Time{Time: time.Now().Add(-time.Hour * 24 * 2)},
				OrganizationID: orgData.ID("string"),
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
		"string":      MakeStrfmtUUID(5, 5),
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

func MakeBranchesMock(n, qty, main int, projID strfmt.UUID) model.Branches {
	isMain := true

	var bs []*model.Branch

	for i := 1; i <= qty; i++ {
		b := &model.Branch{
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

func MakeBranchesBareMock(n int, projID strfmt.UUID) model.Branches {
	bs := MakeBranchesMock(n, 1, 1, projID)
	bs[0].CommitID = nil
	return bs
}

func PtrToString(s string) *string {
	return &s
}
