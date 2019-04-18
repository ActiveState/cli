package mock

import (
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
	hcMock   *hcMock.Mock
	invMock  *invMock.Mock
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate("http://test.tld/"),
		hcMock.Init(),
		invMock.Init(),
		apiMock.Init(),
		authMock.Init(),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
	m.hcMock.Close()
	m.invMock.Close()
	m.apiMock.Close()
	m.authMock.Close()
}

func (m *Mock) MockFullRuntime() {
	m.authMock.MockLoggedin()
	m.apiMock.MockVcsGetCheckpoint()
	m.apiMock.MockSignS3URI()
	m.apiMock.MockGetProject()
	m.invMock.MockOrderRecipes()
	m.invMock.MockPlatforms()

	// Disable the mocking this lib does natively, it's a bad mechanic that has to change, but out of scope for right now
	download.SetMocking(false)
	runtime.InitRequester = m.hcMock.Requester(hcMock.NoOptions)

	m.MockDownload()
}

func (m *Mock) MockDownload() {
	m.httpmock.RegisterWithResponse("GET", "archive.tar.gz", 200, "archive.tar.gz")
}
