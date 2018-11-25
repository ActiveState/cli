package secrets_test

import (
	"encoding/json"
	"io/ioutil"
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
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type SecretsSetCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsSetCommandTestSuite) BeforeTest(suiteName, testName string) {
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

func (suite *SecretsSetCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *SecretsSetCommandTestSuite) TestCommandConfig() {
	cc := secrets.NewCommand(suite.secretsClient).Config().GetCobraCmd().Commands()[0]

	suite.Equal("set", cc.Name())
	suite.Equal("Set the value of a secret", cc.Short, "en-us translation")

	suite.Require().Len(cc.Commands(), 0, "number of subcommands")

	suite.Require().True(cc.HasAvailableFlags())
	suite.NotNil(cc.Flag("project"))
	suite.NotNil(cc.Flag("user"))
}

func (suite *SecretsSetCommandTestSuite) TestExecute_RequiresSecretNameAndValue() {
	cmd := secrets.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"set"})
	err := cmd.Config().Execute()
	suite.EqualError(err, "Argument missing: secrets_set_arg_name_name\nArgument missing: secrets_set_arg_value_name\n")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *SecretsSetCommandTestSuite) TestExecute_FetchOrg_NotAuthenticated() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"set", "secret1", "value1"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertOrgSecret_Succeeds() {
	suite.assertInsertSucceeds("new-org-secret", false, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateOrgSecret_Succeeds() {
	suite.assertInsertSucceeds("org-secret", false, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertProjectSecret_Succeeds() {
	suite.assertInsertSucceeds("new-proj-secret", true, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateProjectSecret_Succeeds() {
	suite.assertInsertSucceeds("proj-secret", true, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertUserSecret_Succeeds() {
	suite.assertInsertSucceeds("new-user-org-secret", false, true)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateUserSecret_Succeeds() {
	suite.assertInsertSucceeds("user-org-secret", false, true)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertUserProjectSecret_Succeeds() {
	suite.assertInsertSucceeds("new-user-proj-secret", true, true)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateUserProjectSecret_Succeeds() {
	suite.assertInsertSucceeds("user-proj-secret", true, true)
}

func (suite *SecretsSetCommandTestSuite) assertInsertSucceeds(secretName string, isProject, isUser bool) {
	bodyChanges := suite.executeSet(secretName, isProject, isUser)
	suite.Require().Len(bodyChanges, 1)
	suite.NotZero(*bodyChanges[0].Value)
	suite.Equal(secretName, *bodyChanges[0].Name)
	suite.Equal(isUser, *bodyChanges[0].IsUser)
	if isProject {
		suite.Equal(strfmt.UUID("00020002-0002-0002-0002-000200020002"), bodyChanges[0].ProjectID)
	} else {
		suite.Zero(bodyChanges[0].ProjectID)
	}
}

func (suite *SecretsSetCommandTestSuite) executeSet(secretName string, isProject, isUser bool) []*models.UserSecretChange {
	cmd := secrets.NewCommand(suite.secretsClient)

	cmdArgs := []string{"set"}
	if isProject {
		cmdArgs = append(cmdArgs, "-p")
	}
	if isUser {
		cmdArgs = append(cmdArgs, "-u")
	}

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	if isProject {
		suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 200)
	}
	suite.secretsMock.RegisterWithCode("GET", "/keypair", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)

	var bodyChanges []*models.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyChanges)
		return 204, "empty-response"
	})

	cmd.Config().GetCobraCmd().SetArgs(append(cmdArgs, secretName, "secret-value"))
	execErr := cmd.Config().Execute()

	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.NoError(failures.Handled())

	return bodyChanges
}

func Test_SecretsSetCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsSetCommandTestSuite))
}
