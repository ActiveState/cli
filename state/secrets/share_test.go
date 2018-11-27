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
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type SecretsShareCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsShareCommandTestSuite) BeforeTest(suiteName, testName string) {
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

func (suite *SecretsShareCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *SecretsShareCommandTestSuite) TestCommandConfig() {
	cc := secrets.NewCommand(suite.secretsClient).Config().GetCobraCmd().Commands()[1]

	suite.Equal("share", cc.Name())
	suite.Equal("Share your organization and project secrets with another user", cc.Short, "en-us translation")

	suite.Require().Len(cc.Commands(), 0, "number of subcommands")
	suite.Require().False(cc.HasAvailableFlags())
}

func (suite *SecretsShareCommandTestSuite) TestExecute_RequiresUserHandle() {
	cmd := secrets.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"share"})
	err := cmd.Config().Execute()
	suite.EqualError(err, "Argument missing: secrets_share_arg_user_name\n")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *SecretsShareCommandTestSuite) TestExecute_FetchOrg_NotAuthenticated() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"share", "scottr"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (suite *SecretsShareCommandTestSuite) TestExecute_FetchOrgMembers_OrgNotFound() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 404)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"share", "scottr"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_org_not_found"))
}

func (suite *SecretsShareCommandTestSuite) TestExecute_FetchOrgMembers_MemberNotFound() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"share", "no-such-user"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("err_api_member_not_found"))
}

func (suite *SecretsShareCommandTestSuite) TestExecute_FetchMemberPublicKey_NotFound() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)
	suite.secretsMock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 404)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"share", "scottr"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, locale.T("keypair_err_publickey_not_found", map[string]string{
		"V0": "scottr",
		"V1": "00020002-0002-0002-0002-000200020002",
	}))
}

func (suite *SecretsShareCommandTestSuite) TestExecute_ShareSuccess() {
	cmd := secrets.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)
	suite.secretsMock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 200)
	suite.secretsMock.RegisterWithCode("GET", "/keypair", 200)

	var bodyChanges []*secretsModels.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets/00020002-0002-0002-0002-000200020002", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyChanges)
		return 204, "empty-response"
	})

	cmd.Config().GetCobraCmd().SetArgs([]string{"share", "scottr"})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.Nil(failures.Handled())

	suite.Require().Len(bodyChanges, 2)

	// assert we can decrypt the changed secrets using the other user's private key
	otherKp, parseFailure := keypairs.ParseRSA("-----BEGIN RSA PRIVATE KEY-----\nMIICXgIBAAKBgQD2d9SU+2dwfirmhsbv6lPBJFeCa/wh6VhLi3VLWDCxqCwz9p62\nfBS4t0Wbjjd2+6FWiCXrGofNXkc9uvv2GhRUB6k/ZKjSkHYDifmQ/llJIeFkdJzn\nqyjzsYovcTAYe4PCQSsIQnIyxZzIpxYpglMyJcuObwn/HRLb0r8ENRCUbQIDAQAB\nAoGBANLq8U0daAPotKXaqNwfV9VtWEYQSxBqNFlR2urDache9pTxdBkOTl1U2Yip\nR+XWqNb4ZBqx9Y1WJPk6zuxonQMuLes8dIjspAnoG1dTN/Lm+cNSuNFtCZAEsOcy\n8Sv7HuDIZcDKnHRxMGFVq8dKCldEVyMWXAro6A58qT9eSV2xAkEA+fVCjN1+hvaT\n366RZEhIeFeYz3MeTRLdSwEEDGrZJTHgteFFzug/zRluTqRzMiDiUn2nbhoeldwx\nPSVz5wJPqwJBAPxs+X2naGh0eDWd1s0Qs2Xer6vTrf1lEXXjCZkiOUYqm5mJSwpK\nzEFu37HnqBTsLPVA+mMdsx2KAAeyhc75dEcCQQD0ualXy8CGmUK8fOkCuzahBHqr\nmXUwVujs92ikU7SYkxYEXTQA2SkmQODcBGx4xvNvenED/nS1mulmiZXJtlyTAkEA\nvsO8aPGzPf2HOz3lr2QHr9zy9fArdWyEHYtPHaN3lUduAEJ5q3WLl4erFk/z/pvd\n/hr1HyK60oAQNcD8zsZG0QJARGfIn630zD8RfqSmHz5YIflAVRmQTOXHvsayo3/+\nH9cW1NPwHp1L8US7gRxkLpqH1XDy9JtaiOt0uJ2KA/pskA==\n-----END RSA PRIVATE KEY-----")
	suite.Require().Nil(parseFailure)

	suite.Equal("finders keepers", suite.decryptSecretValue(otherKp, *bodyChanges[0].Value))
	suite.False(*bodyChanges[0].IsUser)
	suite.Zero(bodyChanges[0].ProjectID)

	suite.Equal("early birds get worms", suite.decryptSecretValue(otherKp, *bodyChanges[1].Value))
	suite.False(*bodyChanges[1].IsUser)
	suite.Equal(strfmt.UUID("00020002-0002-0002-0002-000200020002"), bodyChanges[1].ProjectID)
}

func (suite *SecretsShareCommandTestSuite) decryptSecretValue(kp keypairs.Keypair, value string) string {
	decrBytes, failure := kp.DecodeAndDecrypt(value)
	suite.Require().Nil(failure)
	return string(decrBytes)
}

func Test_SecretsShareCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsShareCommandTestSuite))
}
