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

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *SecretsSetCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *SecretsSetCommandTestSuite) TestCommandConfig() {
	cc := secrets.NewCommand(suite.secretsClient).Config().GetCobraCmd().Commands()[1]

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
	suite.assertSaveSucceeds("new-org-secret", false, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateOrgSecret_Succeeds() {
	suite.assertSaveSucceeds("org-secret", false, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertProjectSecret_Succeeds() {
	suite.assertSaveSucceeds("new-proj-secret", true, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateProjectSecret_Succeeds() {
	suite.assertSaveSucceeds("proj-secret", true, false)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertUserSecret_Succeeds() {
	suite.assertSaveSucceeds("new-user-org-secret", false, true)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateUserSecret_Succeeds() {
	suite.assertSaveSucceeds("user-org-secret", false, true)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_InsertUserProjectSecret_Succeeds() {
	suite.assertSaveSucceeds("new-user-proj-secret", true, true)
}

func (suite *SecretsSetCommandTestSuite) TestExecute_UpdateUserProjectSecret_Succeeds() {
	suite.assertSaveSucceeds("user-proj-secret", true, true)
}

func (suite *SecretsSetCommandTestSuite) assertSaveSucceeds(secretName string, isProject, isUser bool) {
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
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	var userChanges []*models.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &userChanges)
		return 204, "empty-response"
	})

	var sharedChanges []*models.UserSecretShare
	if !isUser {
		// assert secrets get pushed for other users
		suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)
		suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
		suite.secretsMock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 200)
		suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets/00020002-0002-0002-0002-000200020002", func(req *http.Request) (int, string) {
			reqBody, _ := ioutil.ReadAll(req.Body)
			json.Unmarshal(reqBody, &sharedChanges)
			return 204, "empty-response"
		})
	}

	cmd.Config().GetCobraCmd().SetArgs(append(cmdArgs, secretName, "secret-value"))
	execErr := cmd.Config().Execute()

	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.NoError(failures.Handled())

	suite.Require().Len(userChanges, 1)
	suite.NotZero(*userChanges[0].Value)
	suite.Equal(secretName, *userChanges[0].Name)
	suite.Equal(isUser, *userChanges[0].IsUser)
	if isProject {
		suite.Equal(strfmt.UUID("00020002-0002-0002-0002-000200020002"), userChanges[0].ProjectID)
	} else {
		suite.Zero(userChanges[0].ProjectID)
	}

	if !isUser {
		suite.Require().Len(sharedChanges, 1)
		suite.NotZero(*sharedChanges[0].Value)
		suite.Equal(secretName, *sharedChanges[0].Name)
		suite.Equal(userChanges[0].ProjectID, sharedChanges[0].ProjectID)
	} else {
		suite.Require().Len(sharedChanges, 0)
	}

}

func Test_SecretsSetCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsSetCommandTestSuite))
}
