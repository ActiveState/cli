package mock

import (
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
}

func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceHeadChef).String()),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockBuilds() {
	m.httpmock.RegisterWithResponse("POST", "/v1/builds", 201, "builds")
}
