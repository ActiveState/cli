package mock

import (
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String()),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockGetProject() {
	m.httpmock.Register("GET", "/organizations/string/projects/string")
}

func (m *Mock) MockGetProjectDiffCommit() {
	m.httpmock.RegisterWithResponse("GET", "/organizations/string/projects/string", 200, "organizations/string/projects/string-diff-commit")
}

func (m *Mock) MockGetProject404() {
	m.httpmock.RegisterWithCode("GET", "/organizations/string/projects/string", 404)
}
