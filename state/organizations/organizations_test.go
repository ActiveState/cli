package organizations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	Flags.Output = new(string)
	Flags.Verbose = new(bool)
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
	cc := Command.GetCobraCmd()
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = cc.Execute()
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
	cc := Command.GetCobraCmd()
	output := string(commands.JSON)
	Flags.Output = &output
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = cc.Execute()
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
	cc := Command.GetCobraCmd()
	output := string(commands.JSON)
	Flags.Output = &output
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = cc.Execute()
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
	cc := Command.GetCobraCmd()
	output := string(commands.JSON)
	Flags.Output = &output
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = cc.Execute()
	})

	require.NoError(t, outErr)
	require.NoError(t, execErr)
	err := failures.Handled() // Returns an error so have to cast it to a failure
	require.Error(t, err)

	assert.Equal(t, "", outStr, "Expect no output")

	tearDownOrgTest(t, aMock)
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	ex := exiter.New()
	Command.Exiter = ex.Exit
	_, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	// Should not be able to fetch organizations without mock
	handledFail := failures.Handled()
	assert.Error(t, handledFail)
	assert.Contains(t, handledFail.Error(), "no responder found")
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations", 401)
	ex := exiter.New()
	Command.Exiter = ex.Exit
	_, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	handledFail := failures.Handled()
	assert.Error(t, handledFail)
	assert.Contains(t, handledFail.Error(), locale.T("err_api_not_authenticated"))
}

func TestAliases(t *testing.T) {
	cc := Command.GetCobraCmd()
	assert.True(t, cc.HasAlias("orgs"), "Command has alias.")
}
