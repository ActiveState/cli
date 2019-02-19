package secrets_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type SecretsSyncCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsSyncCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")
}

func (suite *SecretsSyncCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *SecretsSyncCommandTestSuite) TestCommandConfig() {
	cc := secrets.NewCommand(suite.secretsClient).Config().GetCobraCmd().Commands()[3]

	suite.Equal("sync", cc.Name())
	suite.Equal("Synchronize your shareable secrets organization-wide", cc.Short, "en-us translation")
	suite.Require().Len(cc.Commands(), 0, "number of subcommands")
	suite.Require().False(cc.HasAvailableFlags())
}

func (suite *SecretsSyncCommandTestSuite) TestExecute_FetchOrg_NotAuthenticated() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (suite *SecretsSyncCommandTestSuite) TestNoDiffForAnyMember() {
	cmd := secrets.NewCommand(suite.secretsClient)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	orgID := "00010001-0001-0001-0001-000100010001"
	scottrID := "00020002-0002-0002-0002-000200020002"
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)
	suite.secretsMock.RegisterWithResponse("GET", fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, scottrID), 404, "notfound")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.Tr("secrets_sync_results_message", "0", "ActiveState"))
}

func (suite *SecretsSyncCommandTestSuite) TestDiffsForSomeMembers() {
	cmd := secrets.NewCommand(suite.secretsClient)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	orgID := "00010001-0001-0001-0001-000100010001"
	scottrID := "00020002-0002-0002-0002-000200020002"
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)

	suite.secretsMock.RegisterWithCode("GET", fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, scottrID), 200)
	var scottrSyncChanges []*models.UserSecretShare
	suite.secretsMock.RegisterWithResponder("PATCH", fmt.Sprintf("/organizations/%s/user_secrets/%s", orgID, scottrID), func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(reqBody, &scottrSyncChanges)
		return 204, "empty-response"
	})

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.Tr("secrets_sync_results_message", "1", "ActiveState"))

	suite.Require().Len(scottrSyncChanges, 2)
	suite.NotZero(*scottrSyncChanges[0].Value)
	suite.Equal("org-secret", *scottrSyncChanges[0].Name)
	suite.Zero(scottrSyncChanges[0].ProjectID)

	suite.NotZero(*scottrSyncChanges[1].Value)
	suite.Equal("proj-secret", *scottrSyncChanges[1].Name)
	suite.Equal(strfmt.UUID("00020002-0002-0002-0002-000200020002"), scottrSyncChanges[1].ProjectID)
}

func (suite *SecretsSyncCommandTestSuite) TestSkipsAuthenticatedUser() {
	cmd := secrets.NewCommand(suite.secretsClient)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	orgID := "00010001-0001-0001-0001-000100010001"
	currentUserID := "00000000-0000-0000-0000-000000000000"
	scottrID := "00020002-0002-0002-0002-000200020002"
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)

	var diffedCurrentUser bool
	suite.secretsMock.RegisterWithResponder("GET", fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, currentUserID), func(req *http.Request) (int, string) {
		diffedCurrentUser = true
		return 500, "needing-this-response-indicates-failure"
	})

	suite.secretsMock.RegisterWithResponse("GET", fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, scottrID), 404, "notfound")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.Tr("secrets_sync_results_message", "0", "ActiveState"))

	suite.False(diffedCurrentUser, "should not have diffed current user")
}

func (suite *SecretsSyncCommandTestSuite) TestFailure_NoLocalPrivateKeyFound() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.T("keypair_err_require_auth"))
}

func (suite *SecretsSyncCommandTestSuite) TestFailure_UnableToDecryptSecret() {
	cmd := secrets.NewCommand(suite.secretsClient)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	orgID := "00010001-0001-0001-0001-000100010001"
	scottrID := "00020002-0002-0002-0002-000200020002"
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)

	diffURI := fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, scottrID)
	suite.secretsMock.RegisterWithResponse("GET", diffURI, 200, diffURI+"-bad-encrypted-secret")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.T("keypairs_err_base64_decoding"))
}

func (suite *SecretsSyncCommandTestSuite) TestFailure_UnableToEncryptNewSecret() {
	cmd := secrets.NewCommand(suite.secretsClient)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	orgID := "00010001-0001-0001-0001-000100010001"
	scottrID := "00020002-0002-0002-0002-000200020002"
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)

	diffURI := fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, scottrID)
	suite.secretsMock.RegisterWithResponse("GET", diffURI, 200, diffURI+"-bad-public-key")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.T("keypairs_err_pem_encoding"))
}

func (suite *SecretsSyncCommandTestSuite) TestFailure_SavingSharedSecrets() {
	cmd := secrets.NewCommand(suite.secretsClient)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	orgID := "00010001-0001-0001-0001-000100010001"
	scottrID := "00020002-0002-0002-0002-000200020002"
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)
	suite.secretsMock.RegisterWithCode("GET", fmt.Sprintf("/organizations/%s/user_secrets/%s/diff", orgID, scottrID), 200)
	suite.secretsMock.RegisterWithResponse("PATCH", fmt.Sprintf("/organizations/%s/user_secrets/%s", orgID, scottrID), 400, "something-happened")

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"sync"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Contains(outStr, locale.T("secrets_err_save"))
}

func Test_SecretsSyncCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsSyncCommandTestSuite))
}
