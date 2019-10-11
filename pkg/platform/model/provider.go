package model

import (
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/gql"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/gqldb/projdb"
)

type ProjectProvider interface {
	gql.ProjectClient
}

var prv = func() ProjectProvider {
	if condition.InTest() {
		return defaultProjectProviderMock()
	}

	endpoint := constants.GraphqlURLStage
	if constants.APIEnv == "prod" {
		endpoint = constants.GraphqlURLProd
	}

	timeout := time.Second * 16

	gc := gqlclient.New(endpoint, nil, timeout)

	p, err := projdb.New(gc)
	if err != nil {
		panic(err)
	}
	return p
}()

func ResetProviderMock() {
	prv = defaultProjectProviderMock()
}

func defaultProjectProviderMock() *projdb.Mock {
	orgData := projdb.MakeOrgDataDefaultMock()

	return projdb.NewMock(
		projdb.NewProjectsRespDefaultMock(orgData),
		orgData,
	)
}

func ProjectProviderMock() *projdb.Mock {
	if !condition.InTest() {
		panic("no")
	}

	mp, ok := prv.(*projdb.Mock)
	if !ok {
		panic("should be available as *projdb.Mock - only during tests")
	}

	return mp
}
