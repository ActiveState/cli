package keypairs_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/stretchr/testify/suite"
)

type KeypairGenerateTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairGenerateTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairGenerateTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairGenerateTestSuite) TestGenerate_Fails_NotEnoughBits() {
	encKeypair, failure := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength-1)
	suite.Require().Nil(encKeypair)
	suite.Equal(keypairs.FailKeypairGenerate, failure.Type)
}

func (suite *KeypairGenerateTestSuite) TestGenerate_NoPassphrase() {
	encKeypair, failure := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)
	suite.Require().NotNil(encKeypair)

	// verify encoded keypair matches generated keypair
	validationKeypair, failure := keypairs.ParseRSA(encKeypair.EncodedPrivateKey)
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, failure := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationPublicKey)

	rsaKey, ok := encKeypair.Keypair.(*keypairs.RSAKeypair)
	suite.Require().True(ok)
	suite.Equal(rsaKey.PublicKey, *validationPublicKey.PublicKey)
}

func (suite *KeypairGenerateTestSuite) TestGenerate_WithPassphrase() {
	encKeypair, failure := keypairs.GenerateEncodedKeypair("tuxedomoon", keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)
	suite.Require().NotNil(encKeypair)

	// verify encoded keypair matches generated keypair with a passphrase
	validationKeypair, failure := keypairs.ParseEncryptedRSA(encKeypair.EncodedPrivateKey, "tuxedomoon")
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, failure := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationPublicKey)

	rsaKey, ok := encKeypair.Keypair.(*keypairs.RSAKeypair)
	suite.Require().True(ok)
	suite.Equal(rsaKey.PublicKey, *validationPublicKey.PublicKey)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Fails_NotEnoughBits() {
	encKeypair, failure := keypairs.GenerateAndSaveEncodedKeypair(suite.secretsClient, "", keypairs.MinimumRSABitLength-1)
	suite.Require().Nil(encKeypair)
	suite.Equal(keypairs.FailKeypairGenerate, failure.Type)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Fails_OnSave() {
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	encKeypair, failure := keypairs.GenerateAndSaveEncodedKeypair(suite.secretsClient, "", keypairs.MinimumRSABitLength)
	suite.Require().Nil(encKeypair)
	suite.Equal(secretsapi.FailKeypairSave, failure.Type)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Success_NoPassphrase() {
	var bodyKeypair *models.KeypairChange
	var bodyErr error
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	encKeypair, failure := keypairs.GenerateAndSaveEncodedKeypair(suite.secretsClient, "", keypairs.MinimumRSABitLength)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(failure)
	suite.Require().NoError(bodyErr)

	// verify encoded keypair matches generated keypair
	validationKeypair, failure := keypairs.ParseRSA(encKeypair.EncodedPrivateKey)
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, failure := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationPublicKey)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Success_WithPassphrase() {
	var bodyKeypair *models.KeypairChange
	var bodyErr error
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	encKeypair, failure := keypairs.GenerateAndSaveEncodedKeypair(suite.secretsClient, "bauhaus", keypairs.MinimumRSABitLength)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(failure)
	suite.Require().NoError(bodyErr)

	// verify encoded keypair matches generated keypair with a passphrase
	validationKeypair, failure := keypairs.ParseEncryptedRSA(encKeypair.EncodedPrivateKey, "bauhaus")
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, failure := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(failure)
	suite.Require().NotNil(validationPublicKey)
}

func Test_KeypairGenerate_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairGenerateTestSuite))
}
