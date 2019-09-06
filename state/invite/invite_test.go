package invite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

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
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func setupHTTPMock(t *testing.T) {

	// For some tests we need to have an activestate.yaml file in our working directory
	root, err := environment.GetRootPath()
	require.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	// we login
	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")
}

func TestSelectOrgRole(t *testing.T) {
	definitions := []struct {
		argValue    string
		promptValue string
		number      OrgRole
	}{
		{"", "owner", Owner},
		{"", "member", Member},
		{"owner", "", Owner},
		{"member", "", Member},
		{"foo", "", None},
	}

	emails := []string{"foo@bar.com", "foo2@bar.com"}

	for _, role := range definitions {
		t.Run(fmt.Sprintf("expect %s(%s)", role.promptValue, role.argValue), func(t *testing.T) {
			pm := pMock.Init()
			pm.On(
				"Select", locale.T("invite_select_org_role", map[string]interface{}{
					"Invitees":     "2 users",
					"Organization": "testOrg",
				}), orgRoleChoices, "",
			).Return(locale.T(fmt.Sprintf("org_role_choice_%s", role.promptValue)), nil)
			orgRole := selectOrgRole(pm, role.argValue, emails, "testOrg")
			require.Equal(t, role.number, orgRole, fmt.Sprintf("orgRole should be %v", role.number))
		})
	}
}

func TestInvite(t *testing.T) {
	setupHTTPMock(t)
	defer httpmock.DeActivate()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "testOrg", "foo@bar.com"})

	// get the organization
	httpmock.Register("GET", "/organizations/testOrg")
	// get the organizations limits
	httpmock.Register("GET", "/organizations/testOrg/limits")

	// then mock an invite call (NOTE: The url will be encoded as a URL string later...)
	httpmock.Register("POST", "/organizations/testOrg/invitations/foo@bar.com")

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
	setupHTTPMock(t)
	defer httpmock.DeActivate()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "testOrgAtLimit", "foo@bar.com,foo2@bar.com"})

	httpmock.Register("GET", "/organizations/testOrgAtLimit")

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
	assert.Contains(t, outStr, "has reached its user limit")
}

func TestCallInParallel(t *testing.T) {
	responseChan := make(chan string, MaxParallelRequests+1)
	args := make([]string, MaxParallelRequests+1)
	for i := range args {
		args[i] = strconv.Itoa(i)
	}

	fails := callInParallel(func(in string) *failures.Failure {
		responseChan <- in
		time.Sleep(30 * time.Millisecond)
		inValue, _ := strconv.Atoi(in)
		if inValue < 5 {
			return failures.FailRuntime.New("test_failure")
		}
		return nil
	}, args)

	close(responseChan)
	require.Len(t, fails, 5, "expected five failures")
	require.Len(t, responseChan, MaxParallelRequests+1)

	// last element should always arrive in the end
	var lastResponse string
	for response := range responseChan {
		lastResponse = response
	}
	require.Equal(t, strconv.FormatInt(MaxParallelRequests, 10), lastResponse, "expected to receive last send element at end")
}

// getTestOrg returns an Organization with specific attributes that we want to test
func getTestOrg(t *testing.T, personal bool, memberCount int, orgName string) *mono_models.Organization {

	var testOrg mono_models.Organization
	var personalString string
	if personal {
		personalString = "true"
	} else {
		personalString = "false"
	}

	err := json.Unmarshal([]byte(fmt.Sprintf(`{
           "organizationID": "11111111-1111-1111-1111-111111111111",
           "added": "1111-11-11T11:11:11.111Z",
           "name": "%s",
           "displayName": "%s",
           "URLname": "%s",
           "personal": %s,
           "owner": true,
           "subscriptionStatus": "active",
           "tier": "string",
           "billingDate": "string",
           "memberCount": %s 
		}`, orgName, orgName, orgName, personalString, strconv.Itoa(memberCount)),
	),
		&testOrg,
	)
	require.NoError(t, err, "could not parse test organization")
	return &testOrg
}

func TestCheckInvites(t *testing.T) {

	t.Run("should fail for personal accounts", func(t *testing.T) {
		org := getTestOrg(t, true, 1, "testOrg")

		err := checkInvites(org, 1)
		require.EqualError(t, err.ToError(), locale.T("invite_personal_org_err"))
	})

	t.Run("fail if organization limits cannot be fetched", func(t *testing.T) {
		setupHTTPMock(t)
		defer httpmock.DeActivate()
		org := getTestOrg(t, false, 1, "nonExistentTestOrg")

		err := checkInvites(org, 1)
		require.EqualError(t, err.ToError(), locale.T("invite_limit_fetch_err"))
	})

	t.Run("fail if organization limits are exceeded", func(t *testing.T) {
		setupHTTPMock(t)
		defer httpmock.DeActivate()
		org := getTestOrg(t, false, 49, "testOrg")
		httpmock.Register("GET", "/organizations/testOrg/limits")

		err := checkInvites(org, 2)

		require.Error(t, err.ToError(), "expected error message due to exceeded user limit")
	})

	t.Run("return true if everything is okay", func(t *testing.T) {
		setupHTTPMock(t)
		defer httpmock.DeActivate()
		org := getTestOrg(t, false, 0, "testOrg")
		httpmock.Register("GET", "/organizations/testOrg/limits")

		err := checkInvites(org, 2)
		require.NoError(t, err.ToError(), "expected no error")
	})
}

func TestAuthError(t *testing.T) {

	setupHTTPMock(t)
	defer httpmock.DeActivate()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "testOrg", "foo@bar.com"})

	// so we are not allowed to get the testOrg limits
	httpmock.RegisterWithCode("GET", "/organizations/testOrg", 401)
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

func TestForbiddenError(t *testing.T) {

	setupHTTPMock(t)
	defer httpmock.DeActivate()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "testOrg", "foo@bar.com"})

	// so we are not allowed to get the testOrg limits
	httpmock.RegisterWithCode("GET", "/organizations/testOrg", 200)
	httpmock.RegisterWithCode("GET", "/organizations/testOrg/limits", 403)
	ex := exiter.New()
	Command.Exiter = ex.Exit
	outStr, outErr := osutil.CaptureStderr(func() {
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})
		require.Equal(t, 1, exitCode, "Exited with code 1")
	})
	require.NoError(t, outErr)

	assert.Contains(t, outStr, locale.T("invite_limit_fetch_err"))
}
