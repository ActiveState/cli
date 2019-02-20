package organizations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

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

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		execErr = Command.Execute()
	})
	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.Error(t, failures.Handled(), "Failure occurred")

	// Should not be able to fetch organizations without mock
	assert.Contains(t, outStr, "no responder found")
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations", 401)
	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		execErr = Command.Execute()
	})
	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.Error(t, failures.Handled(), "Failure occurred")

	assert.Contains(t, outStr, locale.T("err_api_not_authenticated"))
}

func TestAliases(t *testing.T) {
	cc := Command.GetCobraCmd()
	assert.True(t, cc.HasAlias("orgs"), "Command has alias.")
}
