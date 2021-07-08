package organizations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/api"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
}

func setupOrgTest(t *testing.T) *apiMock.Mock {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/tiers")
	authentication.LegacyGet().AuthenticateWithToken("")

	amock := apiMock.Init()
	amock.MockGetOrganizations()
	return amock
}

func tearDownOrgTest(t *testing.T, aMock *apiMock.Mock) {
	httpmock.DeActivate()
	if aMock != nil {
		aMock.Close()
	}
}

func TestOrganizations(t *testing.T) {
	setupOrgTest(t)

	var execErr error
	out := outputhelper.NewCatcher()
	execErr = run(&OrgParams{}, out)
	require.NoError(t, execErr)

	assert.Contains(t, out.CombinedOutput(), "string")

	tearDownOrgTest(t, nil)
}

func TestOrganizationsJSONPaid(t *testing.T) {
	aMock := setupOrgTest(t)
	aMock.MockGetPaidTiers()

	out := outputhelper.NewCatcherByFormat(output.JSONFormatName)
	execErr := run(&OrgParams{}, out)

	require.NoError(t, execErr)

	assert.Equal(t, "[{\"name\":\"string\",\"URLName\":\"string\",\"tier\":\"string\",\"privateProjects\":true}]\x00\n", out.Output(), "Expect privateProjects to be true")

	tearDownOrgTest(t, aMock)
}

func TestOrganizationsJSONFree(t *testing.T) {
	aMock := setupOrgTest(t)
	aMock.MockGetFreeTiers()

	out := outputhelper.NewCatcherByFormat(output.JSONFormatName)
	execErr := run(&OrgParams{}, out)

	require.NoError(t, execErr)

	assert.Equal(t, "[{\"name\":\"string\",\"URLName\":\"string\",\"tier\":\"string\",\"privateProjects\":false}]\x00\n", out.Output(), "Expect privateProjects to be false")

	tearDownOrgTest(t, aMock)
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.LegacyGet().AuthenticateWithToken("")

	err := run(&OrgParams{}, outputhelper.NewCatcher())
	require.Error(t, err)
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.LegacyGet().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations", 401)

	err := run(&OrgParams{}, outputhelper.NewCatcher())
	require.Error(t, err)
}
