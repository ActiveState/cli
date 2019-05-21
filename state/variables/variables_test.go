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

type VariablesCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	authMock      *authMock.Mock
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
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()
}

func (suite *VariablesCommandTestSuite) AfterTest(suiteName, testName string) {
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	httpmock.DeActivate()
	suite.authMock.Close()
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
	suite.Error(execErr, "failure occurred")

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
	suite.Error(execErr, "failure occurred")

	suite.Contains(outStr, locale.T("err_api_project_not_found"))
}

func (suite *VariablesCommandTestSuite) TestExecute_ListAll() {
	cmd := variables.NewCommand(suite.secretsClient)

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

	regexStrFmt := `\b%s\s+%s\s+%v\s+%s\s+%s\s+%s\s*\n`
	suite.Regexp(fmt.Sprintf(regexStrFmt, "DEBUG", "debug", locale.T("variables_value_set"), locale.T("contradiction"), "-", "Local"), outStr)
	suite.Regexp(fmt.Sprintf(regexStrFmt, "PYTHONPATH", "pythonpath", locale.T("variables_value_set"), locale.T("contradiction"), "-", "Local"), outStr)
	suite.Regexp(fmt.Sprintf(regexStrFmt, "org-secret", "org secret", locale.T("variables_value_set"), locale.T("confirmation"), "organization", "Organization"), outStr)
	suite.Regexp(fmt.Sprintf(regexStrFmt, "proj-secret", "proj secret", locale.T("variables_value_set"), locale.T("confirmation"), "organization", "Project"), outStr)
	suite.Regexp(fmt.Sprintf(regexStrFmt, "user-org-secret", "user org secret", locale.T("variables_value_set"), locale.T("confirmation"), "-", "Organization"), outStr)
	suite.Regexp(fmt.Sprintf(regexStrFmt, "user-proj-secret", "user proj secret", locale.T("variables_value_set"), locale.T("confirmation"), "-", "Project"), outStr)
	suite.Regexp(fmt.Sprintf(regexStrFmt, "undefined-org-secret", "undefined org secret", locale.T("variables_value_unset"), locale.T("confirmation"), "organization", "Organization"), outStr)
}

func Test_VariablesCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VariablesCommandTestSuite))
}
