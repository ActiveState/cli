package secrets_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
)

type SecretsCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	secretsClient := secretsapi_test.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *SecretsCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *SecretsCommandTestSuite) TestCommandConfig() {
	cmd := secrets.NewCommand(suite.secretsClient)
	conf := cmd.Config()
	suite.Equal("secrets", conf.Name)
	suite.Equal("secrets_cmd_description", conf.Description, "i18n symbol")

	subCmds := conf.GetCobraCmd().Commands()
	suite.Require().Len(subCmds, 1, "number of subcommands")
	suite.Equal("set", subCmds[0].Name())
	suite.Len(conf.Flags, 0, "number of command flags supported")
	suite.Len(conf.Arguments, 0, "number of commands args supported")
}

func (suite *SecretsCommandTestSuite) TestExecute_FetchOrgNotAuthenticated() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (suite *SecretsCommandTestSuite) TestExecute_FetchProject_NoProjectFound() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects", 404)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_project_not_found"))
}

func (suite *SecretsCommandTestSuite) TestExecute_FetchUserSecrets_NoSecretsFound() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects", 200)
	suite.secretsMock.RegisterWithResponder("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		return 200, "user_secrets-empty"
	})

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("secrets_err_no_secrets_found"))
}

func (suite *SecretsCommandTestSuite) TestExecute_ListAll() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.NoError(failures.Handled(), "unexpected failure occurred")

	suite.Regexp(fmt.Sprintf("\\borg-secret\\s+%s", locale.T("secrets_scope_org")), outStr)
	suite.Regexp(fmt.Sprintf("\\bproj-secret\\s+%s\\s\\(CodeIntel\\)", locale.T("secrets_scope_project")), outStr)
	suite.Regexp(fmt.Sprintf("\\buser-org-secret\\s+%s", locale.T("secrets_scope_user_org")), outStr)
	suite.Regexp(fmt.Sprintf("\\buser-proj-secret\\s+%s\\s\\(TestProj\\)", locale.T("secrets_scope_user_project")), outStr)
	suite.Regexp(fmt.Sprintf("\\buser-proj-secret\\s+%s\\s\\(CodeIntel\\)", locale.T("secrets_scope_user_project")), outStr)
}

func Test_SecretsCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsCommandTestSuite))
}
