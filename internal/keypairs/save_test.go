package keypairs_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/keypairs"
	secrets_models "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/stretchr/testify/suite"
)

type KeypairSaveTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairSaveTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairSaveTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairSaveTestSuite) TestSave_Fails() {
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	encKeypair, failure := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(failure)

	failure = keypairs.SaveEncodedKeypair(suite.secretsClient, encKeypair)
	suite.Equal(secretsapi.FailKeypairSave, failure.Type)
}

func (suite *KeypairSaveTestSuite) TestSave_Succeeds() {
	var bodyKeypair *secrets_models.KeypairChange
	var bodyErr error
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	encKeypair, failure := keypairs.GenerateEncodedKeypair("", keypairs.MinimumRSABitLength)
	suite.Require().NotNil(encKeypair)
	suite.Require().Nil(failure)

	failure = keypairs.SaveEncodedKeypair(suite.secretsClient, encKeypair)
	suite.Require().Nil(failure)
	suite.Require().NoError(bodyErr)

	suite.Equal(encKeypair.EncodedPrivateKey, *bodyKeypair.EncryptedPrivateKey)
	suite.Equal(encKeypair.EncodedPublicKey, *bodyKeypair.PublicKey)
}

func Test_KeypairSave_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairSaveTestSuite))
}
