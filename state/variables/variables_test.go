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

func (suite *VariablesCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	projectfile.Reset()

	lkp := constants.KeypairLocalFileName + ".key"
	err := osutil.CopyTestFileToConfigDir("self-private.key", lkp, 0600)
	suite.Require().NoError(err, "issue creating local private key")

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "state", "variables", "testdata"))

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	activate := httpmock.Activate
	suite.secretsMock = activate(secretsClient.BaseURI)
	suite.platformMock = activate(api.GetServiceURL(api.ServiceMono).String())

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

	suite.platformMock.RegisterWithCode("GET", orgsASPath, 401)

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

	suite.platformMock.RegisterWithCode("GET", orgsASPath, 200)
	retFn := func(req *http.Request) (int, string) {
		// odd requirement for mock framework
		return 200, userSecPath[1:]
	}
	suite.secretsMock.RegisterWithResponder("GET", userSecPath, retFn)
	suite.platformMock.RegisterWithCode("GET", intelPath, 404)

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

	suite.platformMock.RegisterWithCode("GET", orgsASPath, 200)
	suite.platformMock.RegisterWithCode("GET", intelPath, 200)
	retFn := func(req *http.Request) (int, string) {
		// odd requirement for mock framework
		return 200, userSecPath[1:]
	}
	suite.secretsMock.RegisterWithResponder("GET", userSecPath, retFn)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().Nil(failures.Handled(), "unexpected failure occurred")

	rowRegexByColVals := func(vals ...interface{}) string {
		return fmt.Sprintf(
			`\b%s\s+%s\s+%v\s+%s\s+%s\s+%s`,
			vals...,
		)
	}
	rowRegexs := []string{
		rowRegexByColVals(
			"DEBUG",
			"debug",
			locale.T("variables_value_set"),
			"-",
			"-",
			"local",
		),
		rowRegexByColVals(
			"PYTHONPATH",
			"pythonpath",
			locale.T("variables_value_set"),
			"-",
			"-",
			"local",
		),
		rowRegexByColVals(
			"org-secret",
			"org secret",
			locale.T("variables_value_set"),
			locale.T("confirmation"),
			"organization",
			"organization",
		),
		rowRegexByColVals(
			"proj-secret",
			"proj secret",
			locale.T("variables_value_set"),
			locale.T("confirmation"),
			"organization",
			"project",
		),
		rowRegexByColVals(
			"user-org-secret",
			"user org secret",
			locale.T("variables_value_set"),
			locale.T("confirmation"),
			"-",
			"organization",
		),
		rowRegexByColVals(
			"user-proj-secret",
			"user proj secret",
			locale.T("variables_value_set"),
			locale.T("confirmation"),
			"-",
			"project",
		),
		rowRegexByColVals(
			"undefined-org-secret",
			"undefined org secret",
			locale.T("variables_value_unset"),
			locale.T("confirmation"),
			"organization",
			"organization",
		),
	}

	for _, rowRegex := range rowRegexs {
		suite.Regexp(rowRegex, outStr)
	}
}

func Test_VariablesCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VariablesCommandTestSuite))
}
