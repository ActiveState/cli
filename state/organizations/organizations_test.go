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
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
}

func TestOrganizations(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = Command.Execute()
	})
	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.NoError(t, failures.Handled(), "No failure occurred")

	assert.Contains(t, outStr, "test-organization")
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	ex := exiter.New()
	Command.Exiter = ex.Exit
	outStr, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	// Should not be able to fetch organizations without mock
	assert.Contains(t, outStr, "no responder found")
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
	outStr, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	assert.Contains(t, outStr, locale.T("err_api_not_authenticated"))
}

func TestAliases(t *testing.T) {
	cc := Command.GetCobraCmd()
	assert.True(t, cc.HasAlias("orgs"), "Command has alias.")
}
