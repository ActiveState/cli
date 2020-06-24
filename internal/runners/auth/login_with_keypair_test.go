package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type LoginWithKeypairTestSuite struct {
	suite.Suite

	promptMock     *promptMock.Mock
	platformMock   *httpmock.HTTPMock
	secretsapiMock *httpmock.HTTPMock
}

func (suite *LoginWithKeypairTestSuite) BeforeTest(suiteName, testName string) {
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	failures.ResetHandled()

	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	suite.secretsapiMock = httpmock.Activate(secretsapi_test.NewDefaultTestClient("bearing123").BaseURI)

	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	suite.promptMock = promptMock.Init()
	authlet.Prompter = suite.promptMock

	setup(suite.T())
}

func (suite *LoginWithKeypairTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *LoginWithKeypairTestSuite) mockSuccessfulLogin() {
	suite.platformMock.Register("POST", "/login")
	suite.platformMock.Register("GET", "/apikeys")
	suite.platformMock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	suite.platformMock.Register("POST", "/apikeys")
}

func (suite *LoginWithKeypairTestSuite) TestSuccessfulPassphraseMatch() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("foo", nil)

	err := runAuth(&AuthParams{})
	suite.Require().NoError(err, "Executed with error")
	suite.Require().NoError(failures.Handled(), "Unexpected Failure")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// very local keypair is saved
	localKeypair, failure := keypairs.LoadWithDefaults()
	suite.Require().Nil(failure)
	suite.NotNil(localKeypair)
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_HasLocalPrivateKey_MatchesPublicKey() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	suite.secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("bar", nil)

	err := runAuth(&AuthParams{})
	suite.Require().NoError(err, "Executed with error")
	suite.Require().NoError(failures.Handled(), "Unexpected Failure")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, failure := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "bar")
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_NoLocalPrivateKey_OldPasswordMatches() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	suite.secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	// login
	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("bar", nil)
	// passphrase mismatch, prompt for old passphrase
	suite.promptMock.OnMethod("InputSecret").Once().Return("foo", nil)

	err := runAuth(&AuthParams{})
	suite.Require().NoError(err, "Executed with error")
	suite.Require().NoError(failures.Handled(), "Unexpected Failure")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, failure := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "bar")
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_HasMismatchedLocalPrivateKey_OldPasswordMatches() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	osutil.CopyTestFileToConfigDir("mismatched-private.key", constants.KeypairLocalFileName+".key", 0600)

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	suite.secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	// login
	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("bar", nil)
	// passphrase mismatch, prompt for old passphrase
	suite.promptMock.OnMethod("InputSecret").Once().Return("foo", nil)

	err := runAuth(&AuthParams{})
	suite.Require().NoError(err, "Executed with error")
	suite.Require().NoError(failures.Handled(), "Unexpected Failure")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, failure := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "bar")
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)

	// very local keypair is now the new keypair
	localKeypair, failure := keypairs.LoadWithDefaults()
	suite.Require().Nil(failure)
	suite.True(localKeypair.MatchPublicKey(*bodyKeypair.PublicKey))
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_OldPasswordMismatch_GenerateNewKeypair() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	suite.secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	// login
	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("newpassword", nil)
	// passphrase mismatch, prompt for old passphrase
	suite.promptMock.OnMethod("InputSecret").Once().Return("foo", nil)
	// user wants to generate a new keypair
	suite.promptMock.OnMethod("Confirm").Once().Return(true, nil)

	err := runAuth(&AuthParams{})
	suite.Require().NoError(err, "Executed with error")
	suite.Require().NoError(failures.Handled(), "Unexpected Failure")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, failure := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "newpassword")
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)

	// very local keypair is now the new keypair
	localKeypair, failure := keypairs.LoadWithDefaults()
	suite.Require().Nil(failure)
	suite.True(localKeypair.MatchPublicKey(*bodyKeypair.PublicKey))
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_OldPasswordMismatch_DeclineNewKeypair() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	// login
	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("newpassword", nil)
	// passphrase mismatch, prompt for old passphrase
	suite.promptMock.OnMethod("InputSecret").Once().Return("stillwrong", nil)
	// user wants to generate a new keypair
	suite.promptMock.OnMethod("Confirm").Once().Return(true, nil)

	err := runAuth(&AuthParams{})
	suite.Require().Error(err)
	suite.Nil(authentication.ClientAuth(), "Should not have been authenticated")

	// very local keypair does not exist
	localKeypair, failure := keypairs.LoadWithDefaults()
	suite.Require().Equal(keypairs.FailLoadNotFound, failure.Type)
	suite.Nil(localKeypair)
}

func Test_LoginWithKeypair_TestSuite(t *testing.T) {
	suite.Run(t, new(LoginWithKeypairTestSuite))
}
