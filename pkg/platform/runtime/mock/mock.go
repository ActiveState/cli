package mock

import (
	"github.com/ActiveState/cli/internal/download"
	projMock "github.com/ActiveState/cli/internal/projects/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	hcMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/sysinfo"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
	hcMock   *hcMock.Mock
	invMock  *invMock.Mock
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
	projMock *projMock.Mock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate("http://test.tld/"),
		hcMock.Init(),
		invMock.Init(),
		apiMock.Init(),
		authMock.Init(),
		projMock.Init(),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
	m.hcMock.Close()
	m.invMock.Close()
	m.apiMock.Close()
	m.authMock.Close()
	m.projMock.Close()
}

func (m *Mock) MockFullRuntime() {
	m.authMock.MockLoggedin()
	m.apiMock.MockVcsGetCheckpoint()
	m.apiMock.MockSignS3URI()
	m.invMock.MockOrderRecipes()
	m.invMock.MockPlatforms()
	m.projMock.MockGetProject()

	// Disable the mocking this lib does natively, it's a bad mechanic that has to change, but out of scope for right now
	download.SetMocking(false)
	runtime.InitRequester = m.hcMock.Requester(hcMock.NoOptions)

	model.OS = sysinfo.Linux // for now we only support linux, so force it

	m.MockDownload()
}

func (m *Mock) MockDownload() {
	m.httpmock.RegisterWithResponse("GET", "archive.tar.gz", 200, "archive.tar.gz")
}
