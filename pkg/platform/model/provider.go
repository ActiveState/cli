package model

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/gql"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/gqldb/projdb"
	"github.com/go-openapi/strfmt"
)

type ProjectProvider interface {
	gql.ProjectClient
}

var prv = func() ProjectProvider {
	if condition.InTest() {
		orgData := projdb.MakeOrgDataMock()

		return projdb.NewMock(
			projdb.NewProjectsRespMock(orgData),
			orgData,
		)
	}

	endpoint := constants.GraphqlURLStage
	if constants.APIEnv == "prod" {
		endpoint = constants.GraphqlURLProd
	}

	timeout := time.Second * 16

	gc := gqlclient.New(endpoint, nil, timeout)
	fmt.Println(endpoint)

	p, err := projdb.New(gc)
	if err != nil {
		panic(err)
	}
	return p
}()

func AddCommitIDToBranch(id strfmt.UUID, n uint8) {
	mp, ok := prv.(*projdb.Mock)
	if !ok {
		panic("should only ever be run during a test")
	}

	for _, p := range mp.ProjectsResp.Projects {
		for _, b := range p.Branches {
			if b.BranchID == id {
				cid := projdb.MakeStrfmtUUID(n)
				b.CommitID = &cid
			}
		}
	}
}
