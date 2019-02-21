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
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/state/variables"
	"github.com/stretchr/testify/suite"
)

type VariablesCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *VariablesCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	projectfile.Reset()

	err := osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)
	suite.Require().NoError(err, "issue creating local private key")

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "state", "variables", "testdata"))

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
}

func (suite *VariablesCommandTestSuite) AfterTest(suiteName, testName string) {
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	httpmock.DeActivate()
}

func (suite *VariablesCommandTestSuite) TestCommandConfig() {
	cmd := variables.NewCommand(suite.secretsClient)
	conf := cmd.Config()
	suite.Equal("variables", conf.Name)
	suite.Equal("variables_cmd_description", conf.Description, "i18n symbol")

	subCmds := conf.GetCobraCmd().Commands()
	suite.Require().Len(subCmds, 3, "number of subcommands")
	suite.Equal("get", subCmds[0].Name())
	suite.Equal("set", subCmds[1].Name())
	suite.Equal("sync", subCmds[2].Name())
	suite.Len(conf.Flags, 0, "number of command flags supported")
	suite.Len(conf.Arguments, 0, "number of commands args supported")
}

func (suite *VariablesCommandTestSuite) TestExecute_FetchOrgNotAuthenticated() {
	cmd := variables.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (suite *VariablesCommandTestSuite) TestExecute_FetchProject_NoProjectFound() {
	cmd := variables.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.secretsMock.RegisterWithResponder("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		// if we don't do it this way, something with the mock framework breaks
		return 200, "organizations/00010001-0001-0001-0001-000100010001/user_secrets"
	})
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 404)

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_project_not_found"))
}

func (suite *VariablesCommandTestSuite) TestExecute_ListAll() {
	cmd := variables.NewCommand(suite.secretsClient)

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 200)
	suite.secretsMock.RegisterWithResponder("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		// if we don't do it this way, something with the mock framework breaks
		return 200, "organizations/00010001-0001-0001-0001-000100010001/user_secrets"
	})

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().Nil(failures.Handled(), "unexpected failure occurred")

	suite.Regexp(fmt.Sprintf("\\bDEBUG\\s+%v\\s+%s\\s+%s\\s+%s", true, "-", "-", "local"), outStr)
	suite.Regexp(fmt.Sprintf("\\bPYTHONPATH\\s+%s\\s+%s\\s+%s\\s+%s", "%projectDir%/src:%projectDir%/tests", "-", "-", "local"), outStr)
	suite.Regexp(fmt.Sprintf("\\borg-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"), locale.T("confirmation"), "organization", "organization"), outStr)
	suite.Regexp(fmt.Sprintf("\\bproj-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"), locale.T("confirmation"), "organization", "project"), outStr)
	suite.Regexp(fmt.Sprintf("\\buser-org-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"), locale.T("confirmation"), "-", "organization"), outStr)
	suite.Regexp(fmt.Sprintf("\\buser-proj-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret"), locale.T("confirmation"), "-", "project"), outStr)
	suite.Regexp(fmt.Sprintf("\\bundefined-org-secret\\s+%s\\s+%s\\s+%s\\s+%s",
		locale.T("variables_value_secret_undefined"), locale.T("confirmation"), "organization", "organization"), outStr)
}

func Test_VariablesCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VariablesCommandTestSuite))
}
