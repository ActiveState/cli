package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/keypairs"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type LoginWithKeypairTestSuite struct {
	suite.Suite
	cfg configurable

	promptMock     *promptMock.Mock
	platformMock   *httpmock.HTTPMock
	secretsapiMock *httpmock.HTTPMock
}

func (suite *LoginWithKeypairTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.New()
	suite.Require().NoError(err)
	osutil.RemoveConfigFile(suite.cfg.ConfigPath(), constants.KeypairLocalFileName+".key")

	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	suite.secretsapiMock = httpmock.Activate(secretsapi_test.NewDefaultTestClient("bearing123").BaseURI)

	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	suite.promptMock = promptMock.Init()

	setup(suite.T())
}

func (suite *LoginWithKeypairTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *LoginWithKeypairTestSuite) mockSuccessfulLogin() {
	suite.platformMock.Register("POST", "/login")
	suite.platformMock.Register("GET", "/apikeys")
	suite.platformMock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	suite.platformMock.Register("POST", "/apikeys")
	suite.platformMock.Register("GET", "/tiers")
	suite.platformMock.Register("GET", "/organizations/test")
}

func (suite *LoginWithKeypairTestSuite) TestSuccessfulPassphraseMatch() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("foo", nil)

	err := runAuth(&AuthParams{}, suite.promptMock, suite.cfg)
	suite.Require().NoError(err, "Executed with error")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// very local keypair is saved
	localKeypair, err := keypairs.LoadWithDefaults(suite.cfg)
	suite.Require().Nil(err)
	suite.NotNil(localKeypair)
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_HasLocalPrivateKey_MatchesPublicKey() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	osutil.CopyTestFileToConfigDir(suite.cfg.ConfigPath(), "self-private.key", constants.KeypairLocalFileName+".key", 0600)

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	suite.secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	suite.promptMock.OnMethod("Input").Once().Return("testuser", nil)
	suite.promptMock.OnMethod("InputSecret").Once().Return("bar", nil)

	err := runAuth(&AuthParams{}, suite.promptMock, suite.cfg)
	suite.Require().NoError(err, "Executed with error")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, err := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "bar")
	suite.Require().Nil(err)
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

	err := runAuth(&AuthParams{}, suite.promptMock, suite.cfg)
	suite.Require().NoError(err, "Executed with error")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, err := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "bar")
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)
}

func (suite *LoginWithKeypairTestSuite) TestPassphraseMismatch_HasMismatchedLocalPrivateKey_OldPasswordMatches() {
	suite.mockSuccessfulLogin()
	suite.secretsapiMock.Register("GET", "/keypair")

	osutil.CopyTestFileToConfigDir(suite.cfg.ConfigPath(), "mismatched-private.key", constants.KeypairLocalFileName+".key", 0600)

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

	err := runAuth(&AuthParams{}, suite.promptMock, suite.cfg)
	suite.Require().NoError(err, "Executed with error")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, err := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "bar")
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)

	// very local keypair is now the new keypair
	localKeypair, err := keypairs.LoadWithDefaults(suite.cfg)
	suite.Require().Nil(err)
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

	err := runAuth(&AuthParams{}, suite.promptMock, suite.cfg)
	suite.Require().NoError(err, "Executed with error")
	suite.NotNil(authentication.ClientAuth(), "Should have been authenticated")

	// verify encoded keypair matches generated keypair
	suite.Require().NoError(bodyErr)
	suite.Require().NotNil(bodyKeypair)

	validationKeypair, err := keypairs.ParseEncryptedRSA(*bodyKeypair.EncryptedPrivateKey, "newpassword")
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)

	// very local keypair is now the new keypair
	localKeypair, err := keypairs.LoadWithDefaults(suite.cfg)
	suite.Require().Nil(err)
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

	err := runAuth(&AuthParams{}, suite.promptMock, suite.cfg)
	suite.Require().Error(err)
	suite.Nil(authentication.ClientAuth(), "Should not have been authenticated")

	// very local keypair does not exist
	localKeypair, err := keypairs.LoadWithDefaults(suite.cfg)
	suite.Require().Error(err)
	suite.Nil(localKeypair)
}

func Test_LoginWithKeypair_TestSuite(t *testing.T) {
	suite.Run(t, new(LoginWithKeypairTestSuite))
}
