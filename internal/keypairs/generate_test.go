package keypairs_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type KeypairGenerateTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	cfg           keypairs.Configurable
	auth          *authentication.Auth
}

func (suite *KeypairGenerateTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.New()
	suite.Require().NoError(err)
	suite.auth, err = authentication.LegacyGet()
	suite.Require().NoError(err)

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123", suite.auth)
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient
	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairGenerateTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *KeypairGenerateTestSuite) TestGenerate_Fails_NotEnoughBits() {
	encKeypair, err := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength-1)
	suite.Require().Nil(encKeypair)
	suite.Require().Error(err)
}

func (suite *KeypairGenerateTestSuite) TestGenerate_NoPassphrase() {
	encKeypair, err := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)
	suite.Require().NotNil(encKeypair)

	// verify encoded keypair matches generated keypair
	validationKeypair, err := keypairs.ParseRSA(encKeypair.EncodedPrivateKey)
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, err := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(err)
	suite.Require().NotNil(validationPublicKey)

	rsaKey, ok := encKeypair.Keypair.(*keypairs.RSAKeypair)
	suite.Require().True(ok)
	suite.Equal(rsaKey.PublicKey, *validationPublicKey.PublicKey)
}

func (suite *KeypairGenerateTestSuite) TestGenerate_WithPassphrase() {
	encKeypair, err := keypairs.GenerateEncodedKeypair("tuxedomoon", keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)
	suite.Require().NotNil(encKeypair)

	// verify encoded keypair matches generated keypair with a passphrase
	validationKeypair, err := keypairs.ParseEncryptedRSA(encKeypair.EncodedPrivateKey, "tuxedomoon")
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, err := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(err)
	suite.Require().NotNil(validationPublicKey)

	rsaKey, ok := encKeypair.Keypair.(*keypairs.RSAKeypair)
	suite.Require().True(ok)
	suite.Equal(rsaKey.PublicKey, *validationPublicKey.PublicKey)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Fails_NotEnoughBits() {
	encKeypair, err := keypairs.GenerateAndSaveEncodedKeypair(suite.cfg, suite.secretsClient, "", keypairs.MinimumRSABitLength-1, suite.auth)
	suite.Require().Nil(encKeypair)
	suite.Require().Error(err)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Fails_OnSave() {
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	encKeypair, err := keypairs.GenerateAndSaveEncodedKeypair(suite.cfg, suite.secretsClient, "", keypairs.MinimumRSABitLength, suite.auth)
	suite.Require().Nil(encKeypair)
	suite.Require().Error(err)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Success_NoPassphrase() {
	var bodyKeypair *secrets_models.KeypairChange
	var bodyErr error
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := io.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	encKeypair, err := keypairs.GenerateAndSaveEncodedKeypair(suite.cfg, suite.secretsClient, "", keypairs.MinimumRSABitLength, suite.auth)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(err)
	suite.Require().NoError(bodyErr)

	// verify encoded keypair matches generated keypair
	validationKeypair, err := keypairs.ParseRSA(encKeypair.EncodedPrivateKey)
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, err := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(err)
	suite.Require().NotNil(validationPublicKey)
}

func (suite *KeypairGenerateTestSuite) TestGenerateAndSave_Success_WithPassphrase() {
	var bodyKeypair *secrets_models.KeypairChange
	var bodyErr error
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := io.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	encKeypair, err := keypairs.GenerateAndSaveEncodedKeypair(suite.cfg, suite.secretsClient, "bauhaus", keypairs.MinimumRSABitLength, suite.auth)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(err)
	suite.Require().NoError(bodyErr)

	// verify encoded keypair matches generated keypair with a passphrase
	validationKeypair, err := keypairs.ParseEncryptedRSA(encKeypair.EncodedPrivateKey, "bauhaus")
	suite.Require().Nil(err)
	suite.Require().NotNil(validationKeypair)
	suite.Equal(encKeypair.Keypair, validationKeypair)

	// verify encoded public key matches generated keypair's public key
	validationPublicKey, err := keypairs.ParseRSAPublicKey(encKeypair.EncodedPublicKey)
	suite.Require().Nil(err)
	suite.Require().NotNil(validationPublicKey)
}

func Test_KeypairGenerate_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairGenerateTestSuite))
}
