package mock

import (
	rt "runtime"

	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
)

type Mock struct {
	httpmock  *httpmock.HTTPMock
	hcMock    *hcMock.Mock
	invMock   *invMock.Mock
	apiMock   *apiMock.Mock
	GraphMock *graphMock.Mock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate("http://test.tld/"),
		hcMock.Init(),
		invMock.Init(),
		apiMock.Init(),
		graphMock.Init(),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
	m.hcMock.Close()
	m.invMock.Close()
	m.apiMock.Close()
	m.GraphMock.Close()
}

func (m *Mock) MockFullRuntime() {
	m.apiMock.MockSignS3URI()
	m.invMock.MockOrderRecipes()
	m.invMock.MockPlatforms()
	m.invMock.MockSolutions()
	m.GraphMock.ProjectByOrgAndName(graphMock.NoOptions)
	m.GraphMock.Checkpoint(graphMock.NoOptions)
	m.hcMock.MockBuilds(hcMock.Completed)

	// Disable the mocking this lib does natively, it's a bad mechanic that has to change, but out of scope for right now
	download.SetMocking(false)

	m.MockCamelDownload()
}

func (m *Mock) MockCamelDownload() {
	switch rt.GOOS {
	case "darwin":
		m.httpmock.RegisterWithResponse("GET", "python.tar.gz", 200, "python-macos.tar.gz")
		m.httpmock.RegisterWithResponse("GET", "legacy-python.tar.gz", 200, "legacy-python-macos.tar.gz")
	case "windows":
		m.httpmock.RegisterWithResponse("GET", "python.zip", 200, "python.zip")
		m.httpmock.RegisterWithResponse("GET", "legacy-python.zip", 200, "legacy-python.zip")
	default:
		m.httpmock.RegisterWithResponse("GET", "python.tar.gz", 200, "python.tar.gz")
		m.httpmock.RegisterWithResponse("GET", "legacy-python.tar.gz", 200, "legacy-python.tar.gz")
	}
}
