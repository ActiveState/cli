package mock

import (
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type Mock struct {
	httpmock  *httpmock.HTTPMock
	hcMock    *hcMock.Mock
	invMock   *invMock.Mock
	apiMock   *apiMock.Mock
	authMock  *authMock.Mock
	GraphMock *graphMock.Mock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate("http://test.tld/"),
		hcMock.Init(),
		invMock.Init(),
		apiMock.Init(),
		authMock.Init(),
		graphMock.Init(),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
	m.hcMock.Close()
	m.invMock.Close()
	m.apiMock.Close()
	m.authMock.Close()
	m.GraphMock.Close()
}

func (m *Mock) MockFullRuntime() {
	m.authMock.MockLoggedin()
	m.apiMock.MockSignS3URI()
	m.invMock.MockOrderRecipes()
	m.invMock.MockPlatforms()
	m.GraphMock.ProjectByOrgAndName(graphMock.NoOptions)
	m.GraphMock.Checkpoint(graphMock.NoOptions)

	// Disable the mocking this lib does natively, it's a bad mechanic that has to change, but out of scope for right now
	download.SetMocking(false)
	runtime.InitRequester = m.hcMock.Requester(hcMock.NoOptions)

	m.MockDownload()
}

func (m *Mock) MockDownload() {
	m.httpmock.RegisterWithResponse("GET", "python"+runtime.InstallerExtension, 200, "python"+runtime.InstallerExtension)
	m.httpmock.RegisterWithResponse("GET", "legacy-python"+runtime.InstallerExtension, 200, "legacy-python"+runtime.InstallerExtension)
}
