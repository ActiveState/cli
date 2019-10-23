package mock

import (
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

type ResponseType int

const (
	Started ResponseType = iota
	Failed
	Completed
	RunFail
	RunFailMalformed
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

func (m *Mock) MockBuilds(respType ResponseType) {
	regWithResp := m.httpmock.RegisterWithResponse
	path := "/v1/builds"

	switch respType {
	case Started:
		regWithResp("POST", path, 202, "builds-started")
	case Failed:
		regWithResp("POST", path, 201, "builds-failed")
	case Completed:
		regWithResp("POST", path, 201, "builds-completed")
	case RunFail:
		m.httpmock.RegisterWithResponseBody("POST", path, 500, `{"message": "no"}`)
	case RunFailMalformed:
		m.httpmock.RegisterWithResponseBody("POST", path, 201, `{"type": "no"}`)
	default:
		panic("use a valid ResponseType constant")
	}
}
