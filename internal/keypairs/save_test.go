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

type KeypairSaveTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	cfg           keypairs.Configurable
	auth          *authentication.Auth
}

func (suite *KeypairSaveTestSuite) BeforeTest(suiteName, testName string) {
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

func (suite *KeypairSaveTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *KeypairSaveTestSuite) TestSave_Fails() {
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	encKeypair, err := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(err)

	err = keypairs.SaveEncodedKeypair(suite.cfg, suite.secretsClient, encKeypair, suite.auth)
	suite.Error(err)
}

func (suite *KeypairSaveTestSuite) TestSave_Succeeds() {
	var bodyKeypair *secrets_models.KeypairChange
	var bodyErr error
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := io.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	encKeypair, err := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(err)

	err = keypairs.SaveEncodedKeypair(suite.cfg, suite.secretsClient, encKeypair, suite.auth)
	suite.Require().Nil(err)
	suite.Require().NoError(bodyErr)

	suite.Equal(encKeypair.EncodedPrivateKey, *bodyKeypair.EncryptedPrivateKey)
	suite.Equal(encKeypair.EncodedPublicKey, *bodyKeypair.PublicKey)
}

func Test_KeypairSave_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairSaveTestSuite))
}
