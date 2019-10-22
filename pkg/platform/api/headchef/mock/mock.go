package mock

import (
	"runtime"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
}

func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceHeadChef).String())
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockBuilds() {
	m.httpmock.RegisterWithResponse("POST", "/v1/builds", 201, "builds")
}
