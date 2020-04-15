package invite

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	pMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
)

type InviteTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
	ex       *exiter.Exiter
}

// Run provides suite functionality around golang subtests.
//
// It is not in the vendor packaged source for stretchr/testify Suite, but is in v1.3.0.
// Until we upgrade, we shim it.
func (s *InviteTestSuite) Run(name string, subtest func()) bool {
	oldT := s.T()
	defer s.SetT(oldT)
	return oldT.Run(name, func(t *testing.T) {
		s.SetT(t)
		subtest()
	})
}

func (s *InviteTestSuite) BeforeTest(suiteName, testName string) {
	s.apiMock = apiMock.Init()
	s.authMock = authMock.Init()
	// For some tests we need to have an activestate.yaml file in our working directory
	root, err := environment.GetRootPath()
	s.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	// mock that we are logged in
	s.authMock.MockLoggedin()

	s.ex = exiter.New()
	Command.Exiter = s.ex.Exit

}

func (s *InviteTestSuite) AfterTest(suiteName, testName string) {
	s.apiMock.Close()
	s.authMock.Close()
}

func (s *InviteTestSuite) TestSelectOrgRole() {
	definitions := []struct {
		argValue    string
		promptValue string
		number      OrgRole
		noError     bool
	}{
		{"", "owner", Owner, true},
		{"", "member", Member, true},
		{"owner", "", Owner, true},
		{"member", "", Member, true},
		{"foo", "", None, false},
	}

	emails := []string{"foo@bar.com", "foo2@bar.com"}

	for _, role := range definitions {
		s.Run(fmt.Sprintf("expect %s(%s)", role.promptValue, role.argValue), func() {
			choices, _ := orgRoleChoices()
			pm := pMock.Init()
			pm.On(
				"Select", locale.T("invite_select_org_role", map[string]interface{}{
					"Invitees":     fmt.Sprintf("2 %s", locale.T("users_plural")),
					"Organization": "testOrg",
				}), choices, "",
			).Return(locale.T(fmt.Sprintf("org_role_choice_%s", role.promptValue)), nil)
			orgRole, fail := selectOrgRole(pm, role.argValue, emails, "testOrg")
			if role.noError {
				s.NoError(fail.ToError(), "no error expected")
			} else {
				s.Error(fail.ToError(), "expected an error")
			}
			s.Equal(role.number, orgRole, fmt.Sprintf("orgRole should be %v", role.number))
		})
	}
}

func (s *InviteTestSuite) TestInvite() {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "string", "foo@bar.com"})

	// get the organization
	s.apiMock.MockGetOrganization()
	s.apiMock.MockGetOrganizationLimits()
	s.apiMock.MockInviteUser()

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = Command.Execute()
	})

	s.NoError(outErr)
	s.NoError(execErr)
	s.NoError(failures.Handled(), "No failure occurred")

	s.Contains(outStr, "foo@bar.com")
}

func (s *InviteTestSuite) TestInviteUserLimit() {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "string", "foo@bar.com,foo2@bar.com"})

	s.apiMock.MockGetOrganization()
	s.apiMock.MockGetOrganizationLimitsReached()

	_, outErr := osutil.CaptureStderr(func() {
		exitCode := s.ex.WaitForExit(func() {
			Command.Execute()
		})
		s.Equal(1, exitCode, "Exited with code 1")
	})
	s.NoError(outErr)

	// Should not be able to invite users without mock
	handledFail := failures.Handled()
	s.Error(handledFail)
	s.Contains(handledFail.Error(), "The request exceeds the limit")
}

func (s *InviteTestSuite) TestCallInParallel() {
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
	s.Len(fails, 5, "expected five failures")
	s.Len(responseChan, MaxParallelRequests+1)

	// last element should always arrive in the end
	var lastResponse string
	for response := range responseChan {
		lastResponse = response
	}
	s.Equal(strconv.FormatInt(MaxParallelRequests, 10), lastResponse, "expected to receive last send element at end")
}

// getTestOrg returns an Organization with specific attributes that we want to test
func (s *InviteTestSuite) getTestOrg(personal bool, memberCount int, orgName string) *mono_models.Organization {
	added, _ := time.Parse(time.RFC1123, "1970-01-01T00:00:00Z")
	return &mono_models.Organization{
		URLname:        orgName,
		Name:           orgName,
		Personal:       personal,
		AddOns:         nil,
		BillingDate:    nil,
		MemberCount:    int64(memberCount),
		Owner:          true,
		Added:          strfmt.DateTime(added),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
	}
}

func (s *InviteTestSuite) TestIsInvitationPossible() {

	s.Run("should fail for personal accounts", func() {
		org := s.getTestOrg(true, 1, "string")

		err := isInvitationPossible(org, 1)
		s.EqualError(err.ToError(), locale.T("invite_personal_org_err"))
	})

	s.Run("fail if organization limits cannot be fetched", func() {
		org := s.getTestOrg(false, 1, "nonExistentTestOrg")

		err := isInvitationPossible(org, 1)
		s.EqualError(err.ToError(), locale.T("invite_limit_fetch_err"))
	})

	s.Run("fail if organization limits are exceeded", func() {
		org := s.getTestOrg(false, 49, "string")

		s.apiMock.MockGetOrganizationLimits()

		err := isInvitationPossible(org, 2)

		s.Error(err.ToError(), "expected error message due to exceeded user limit")
	})

	s.Run("return true if everything is okay", func() {
		org := s.getTestOrg(false, 0, "string")

		s.apiMock.MockGetOrganizationLimits()

		err := isInvitationPossible(org, 2)
		s.NoError(err.ToError(), "expected no error")
	})
}

func (s *InviteTestSuite) TestAuthError() {

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "string", "foo@bar.com"})

	// ... so we are not authorized to get the testOrg
	s.apiMock.MockGetOrganization401()

	_, outErr := osutil.CaptureStderr(func() {
		exitCode := s.ex.WaitForExit(func() {
			Command.Execute()
		})
		s.Equal(1, exitCode, "Exited with code 1")
	})
	s.NoError(outErr)

	handledFail := failures.Handled()
	s.Error(handledFail)
	s.Contains(handledFail.Error(), locale.T("err_api_not_authenticated"))
}

func (s *InviteTestSuite) TestForbiddenError() {

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--role", "member", "--organization", "string", "foo@bar.com"})

	// we are not allowed to get the testOrg limits
	s.apiMock.MockGetOrganization()
	s.apiMock.MockGetOrganizationLimits403()
	_, outErr := osutil.CaptureStderr(func() {
		exitCode := s.ex.WaitForExit(func() {
			Command.Execute()
		})
		s.Equal(1, exitCode, "Exited with code 1")
	})
	s.NoError(outErr)

	handledFail := failures.Handled()
	s.Error(handledFail)
	s.Contains(handledFail.Error(), locale.T("invite_limit_fetch_err"))
}

func TestInviteSuite(t *testing.T) {
	suite.Run(t, new(InviteTestSuite))
}
