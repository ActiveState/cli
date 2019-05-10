package variables_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/state/variables"
	"github.com/stretchr/testify/suite"
)

const (
	orgsPath    = "/organizations"
	userSecPath = orgsPath + "/00010001-0001-0001-0001-000100010001/user_secrets"
	orgsASPath  = orgsPath + "/ActiveState"
	intelPath   = orgsASPath + "/projects/CodeIntel"
)

type VariablesCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	authMock      *authMock.Mock
}

func (st *VariablesCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	projectfile.Reset()

	lkp := constants.KeypairLocalFileName + ".key"
	err := osutil.CopyTestFileToConfigDir("self-private.key", lkp, 0600)
	st.Require().NoError(err, "issue creating local private key")

	// support test projectfile access
	root, err := environment.GetRootPath()
	st.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "state", "variables", "testdata"))

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	st.Require().NotNil(secretsClient)
	st.secretsClient = secretsClient

	activate := httpmock.Activate
	st.secretsMock = activate(secretsClient.BaseURI)
	st.platformMock = activate(api.GetServiceURL(api.ServiceMono).String())

	st.authMock = authMock.Init()
	st.authMock.MockLoggedin()
}

func (st *VariablesCommandTestSuite) AfterTest(suiteName, testName string) {
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	httpmock.DeActivate()
	st.authMock.Close()
}

func (st *VariablesCommandTestSuite) TestCommandConfig() {
	cmd := variables.NewCommand(st.secretsClient)
	conf := cmd.Config()
	st.Equal("variables", conf.Name)
	st.Equal("variables_cmd_description", conf.Description, "i18n symbol")

	subCmds := conf.GetCobraCmd().Commands()
	st.Require().Len(subCmds, 3, "number of subcommands")
	st.Equal("get", subCmds[0].Name())
	st.Equal("set", subCmds[1].Name())
	st.Equal("sync", subCmds[2].Name())
	st.Len(conf.Flags, 0, "number of command flags supported")
	st.Len(conf.Arguments, 0, "number of commands args supported")
}

func (st *VariablesCommandTestSuite) TestExecute_FetchOrgNotAuthenticated() {
	cmd := variables.NewCommand(st.secretsClient)

	st.platformMock.RegisterWithCode("GET", orgsASPath, 401)

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	st.Require().NoError(outErr)
	st.Error(execErr, "failure occurred")

	st.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (st *VariablesCommandTestSuite) TestExecute_FetchProject_NoProjectFound() {
	cmd := variables.NewCommand(st.secretsClient)

	st.platformMock.RegisterWithCode("GET", orgsASPath, 200)
	retFn := func(req *http.Request) (int, string) {
		// odd requirement for mock framework
		return 200, userSecPath[1:]
	}
	st.secretsMock.RegisterWithResponder("GET", userSecPath, retFn)
	st.platformMock.RegisterWithCode("GET", intelPath, 404)

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	st.Require().NoError(outErr)
	st.Error(execErr, "failure occurred")

	st.Contains(outStr, locale.T("err_api_project_not_found"))
}

func (st *VariablesCommandTestSuite) TestExecute_ListAll() {
	cmd := variables.NewCommand(st.secretsClient)

	st.platformMock.RegisterWithCode("GET", orgsASPath, 200)
	st.platformMock.RegisterWithCode("GET", intelPath, 200)
	retFn := func(req *http.Request) (int, string) {
		// odd requirement for mock framework
		return 200, userSecPath[1:]
	}
	st.secretsMock.RegisterWithResponder("GET", userSecPath, retFn)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	st.Require().NoError(outErr)
	st.Require().NoError(execErr)
	st.Require().Nil(failures.Handled(), "unexpected failure occurred")

	spf := fmt.Sprintf
	st.Regexp(spf("\\bDEBUG\\s+%v\\s+%s\\s+%s\\s+%s",
		true,
		"-",
		"-",
		"local",
	), outStr)
	st.Regexp(spf("\\bPYTHONPATH\\s+%s\\s+%s\\s+%s\\s+%s",
		"%projectDir%/src:%projectDir%/tests",
		"-",
		"-",
		"local",
	), outStr)
	st.Regexp(spf("\\borg-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"),
		locale.T("confirmation"),
		"organization",
		"organization",
	), outStr)
	st.Regexp(spf("\\bproj-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"),
		locale.T("confirmation"),
		"organization",
		"project",
	), outStr)
	st.Regexp(spf(
		"\\buser-org-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"),
		locale.T("confirmation"),
		"-",
		"organization",
	), outStr)
	st.Regexp(spf(
		"\\buser-proj-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"),
		locale.T("confirmation"),
		"-",
		"project",
	), outStr)
	st.Regexp(spf(
		"\\bundefined-org-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret_undefined"),
		locale.T("confirmation"),
		"organization",
		"organization",
	), outStr)
}

func Test_VariablesCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VariablesCommandTestSuite))
}
