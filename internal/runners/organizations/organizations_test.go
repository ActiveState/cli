package organizations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
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
	authentication.Get().AuthenticateWithToken("")

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
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = run(&OrgParams{})
	})
	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.NoError(t, failures.Handled(), "No failure occurred")

	assert.Contains(t, outStr, "string")

	tearDownOrgTest(t, nil)
}

func TestOrganizationsJSONPaid(t *testing.T) {
	aMock := setupOrgTest(t)
	aMock.MockGetPaidTiers()

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = run(&OrgParams{Output: output.FormatJSON})
	})

	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.NoError(t, failures.Handled(), "No failure occurred")

	assert.Equal(t, "[{\"name\":\"string\",\"URLName\":\"string\",\"tier\":\"string\",\"privateProjects\":true}]\n", outStr, "Expect privateProjects to be true")

	tearDownOrgTest(t, aMock)
}

func TestOrganizationsJSONFree(t *testing.T) {
	aMock := setupOrgTest(t)
	aMock.MockGetFreeTiers()

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = run(&OrgParams{Output: output.FormatJSON})
	})

	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.NoError(t, failures.Handled(), "No failure occurred")

	assert.Equal(t, "[{\"name\":\"string\",\"URLName\":\"string\",\"tier\":\"string\",\"privateProjects\":false}]\n", outStr, "Expect privateProjects to be false")

	tearDownOrgTest(t, aMock)
}

func TestOrganizationsJSONBad(t *testing.T) {
	aMock := setupOrgTest(t)
	aMock.MockGetBadTiers()

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = run(&OrgParams{Output: output.FormatJSON})
	})

	require.Error(t, execErr)
	require.NoError(t, outErr)
	assert.Equal(t, "", outStr, "Expect no output")

	tearDownOrgTest(t, aMock)
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	err := run(&OrgParams{})
	require.Error(t, err)
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations", 401)

	err := run(&OrgParams{})
	require.Error(t, err)
}
