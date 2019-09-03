package invite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	pMock "github.com/ActiveState/cli/internal/prompt/mock"
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

func TestSelectOrgRole(t *testing.T) {
	definitions := []struct {
		stringValue string
		number      OrgRole
	}{{"owner", Owner}}

	for _, role := range definitions {
		t.Run(fmt.Sprintf("expect %s", role.stringValue), func(t *testing.T) {
			pm := pMock.Init()
			pm.On(
				"Select", locale.T("invite_select_org_role", 2), orgRoleChoices, "",
			).Return(locale.T(fmt.Sprintf("org_role_choice_%s", role.stringValue)), nil)
			orgRole := selectOrgRole(pm, 2)
			require.Equal(t, role.number, orgRole, fmt.Sprintf("orgRole should be %s", role.stringValue))
		})
	}
}

func TestSendInvite(t *testing.T) {

}

func TestInvite(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	// log in first
	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	// then get the organizations limits
	httpmock.Register("GET", "/organizations/testOrg/limits")

	// then mock an invite call
	httpmock.Register("POST", "/organizations/testOrg/invitations/foo%%40bar.com")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = Command.Execute()
	})
	require.NoError(t, outErr)
	require.NoError(t, execErr)
	assert.NoError(t, failures.Handled(), "No failure occurred")

	assert.Contains(t, outStr, "foo@bar.com")
}

func TestInviteUserLimit(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	httpmock.Register("GET", "/organizations/testOrgAtLimit/limits")

	ex := exiter.New()
	Command.Exiter = ex.Exit
	outStr, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	// Should not be able to invite users without mock
	assert.Contains(t, outStr, "has reached user limit")
}

// return with exit code 1 if
func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	// we login
	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	// ... but then do not mock the following requests

	// ... so that we will fail
	ex := exiter.New()
	Command.Exiter = ex.Exit
	outStr, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	// Should not be able to invite users without mock
	assert.Contains(t, outStr, "no responder found")
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	// invalidate the login token
	authentication.Get().AuthenticateWithToken("")

	// so we are not allowed to get the testOrg limits
	httpmock.RegisterWithCode("GET", "/organizations/testOrg/limits", 401)
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
