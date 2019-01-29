package keypair_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/stretchr/testify/suite"
)

type KeypairGenerateTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairGenerateTestSuite) BeforeTest(suiteName, testName string) {
	// reset flags and failures
	failures.ResetHandled()
	keypair.Flags.Bits = constants.DefaultRSABitLength
	keypair.Flags.DryRun = false
	keypair.Flags.SkipPassphrase = false

	secretsClient := secretsapi_test.InitializeTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairGenerateTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairGenerateTestSuite) TestExecute_SavesNewKeypair() {
	cmd := keypair.Command

	var bodyKeypair *models.Keypair
	var bodyErr error
	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "keypair"
	})

	var execErr error
	outStr, _ := osutil.CaptureStdout(func() {
		cmd.GetCobraCmd().SetArgs([]string{"generate", "-b", "512"})
		osutil.WrapStdin(func() { execErr = cmd.Execute() }, "abc123")
	})

	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.Require().NoError(failures.Handled(), "unexpected failure occurred")

	suite.Contains(*bodyKeypair.EncryptedPrivateKey, "RSA PRIVATE KEY")
	suite.Contains(*bodyKeypair.EncryptedPrivateKey, "ENCRYPTED")
	suite.Contains(*bodyKeypair.PublicKey, "RSA PUBLIC KEY")
	suite.Contains(outStr, "Keypair generated successfully")

	keyContents, fileErr := osutil.ReadConfigFile("private.key")
	suite.Require().NoError(fileErr)
	suite.Contains(keyContents, "RSA PRIVATE KEY")
	suite.NotContains(keyContents, "ENCRYPTED")
	suite.NotContains(keyContents, "RSA PUBLIC KEY")
}

func (suite *KeypairGenerateTestSuite) TestExecute_SaveFails() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	var execErr error
	osutil.CaptureStdout(func() {
		cmd.GetCobraCmd().SetArgs([]string{"generate", "-b", "512"})
		osutil.WrapStdin(func() { execErr = cmd.Execute() }, "abc123")
	})

	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")
	failure := failures.Handled().(*failures.Failure)
	suite.True(failure.Type.Matches(secretsapi.FailSave), "should be a FailSave failure")
}

func (suite *KeypairGenerateTestSuite) TestExecute_DryRun() {
	cmd := keypair.Command

	var execErr error
	outStr, _ := osutil.CaptureStdout(func() {
		cmd.GetCobraCmd().SetArgs([]string{"generate", "-b", "512", "--dry-run"})
		osutil.WrapStdin(func() { execErr = cmd.Execute() }, "abc123")
	})
	suite.Require().NoError(execErr)
	suite.Require().NoError(failures.Handled(), "is a failure")

	suite.Contains(outStr, "RSA PRIVATE KEY")
	suite.Contains(outStr, "RSA PUBLIC KEY")
}

func Test_KeypairGenerateTestSuite(t *testing.T) {
	suite.Run(t, new(KeypairGenerateTestSuite))
}
