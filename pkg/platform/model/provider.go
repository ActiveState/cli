package model

import (
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/platform/api/client"
	"github.com/ActiveState/cli/internal/platform/api/graphql/projclient"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type ProjectProvider interface {
	client.ProjectProvider
}

var prv = func() ProjectProvider {
	if condition.InTest() {
		return defaultProjectProviderMock()
	}

	endpoint := api.GetServiceURL(api.ServiceGraphQL)
	timeout := time.Second * 16

	gc := gqlclient.New(endpoint.String(), nil, authentication.Get(), timeout)

	p, err := projclient.New(gc)
	if err != nil {
		panic(err)
	}
	return p
}()

func ResetProviderMock() {
	prv = defaultProjectProviderMock()
}

func defaultProjectProviderMock() *projclient.Mock {
	orgData := projclient.MakeOrgDataDefaultMock()

	return projclient.NewMock(
		projclient.NewProjectsRespDefaultMock(orgData),
		orgData,
	)
}

func ProjectProviderMock() *projclient.Mock {
	if !condition.InTest() {
		panic("no")
	}

	mp, ok := prv.(*projclient.Mock)
	if !ok {
		panic("should be available as *projclient.Mock - only during tests")
	}

	return mp
}
