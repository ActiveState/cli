package keypairs_test

import (
	"net/http"
	"testing"

	apiModels "github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type KeypairFetcherTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairFetcherTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	secretsClient := secretsapi_test.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairFetcherTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairFetcherTestSuite) TestFetch_NotFound() {
	httpmock.RegisterWithCode("GET", "/keypair", 404)
	kp, failure := keypairs.Fetch(suite.secretsClient)
	suite.Nil(kp)
	suite.True(failure.Type.Matches(secretsapi.FailNotFound))
}

func (suite *KeypairFetcherTestSuite) TestFetch_ErrorParsing() {
	httpmock.RegisterWithResponder("GET", "/keypair", func(req *http.Request) (int, string) {
		return 200, "keypair-unparseable"
	})

	kp, failure := keypairs.Fetch(suite.secretsClient)
	suite.Nil(kp)
	suite.True(failure.Type.Matches(keypairs.FailKeypair))
}

func (suite *KeypairFetcherTestSuite) TestFetch_Success() {
	httpmock.RegisterWithCode("GET", "/keypair", 200)
	kp, failure := keypairs.Fetch(suite.secretsClient)
	suite.Require().Nil(failure)
	suite.IsType(&keypairs.RSAKeypair{}, kp)
}

func (suite *KeypairFetcherTestSuite) TestFetchRaw_NotFound() {
	httpmock.RegisterWithCode("GET", "/keypair", 404)
	kp, failure := keypairs.FetchRaw(suite.secretsClient)
	suite.Nil(kp)
	suite.True(failure.Type.Matches(secretsapi.FailNotFound))
}

func (suite *KeypairFetcherTestSuite) TestFetchRaw_Success() {
	httpmock.RegisterWithCode("GET", "/keypair", 200)
	kp, failure := keypairs.FetchRaw(suite.secretsClient)
	suite.Require().Nil(failure)
	suite.IsType(&models.Keypair{}, kp)
}

func (suite *KeypairFetcherTestSuite) TestFetchPublicKey_NotFound() {
	httpmock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 404)
	kp, failure := keypairs.FetchPublicKey(suite.secretsClient, &apiModels.User{
		UserID: strfmt.UUID("00020002-0002-0002-0002-000200020002"),
	})
	suite.Nil(kp)
	suite.True(failure.Type.Matches(secretsapi.FailNotFound))
}

func (suite *KeypairFetcherTestSuite) TestFetchPublicKey_ErrorParsing() {
	httpmock.RegisterWithResponder("GET", "/publickeys/00020002-0002-0002-0002-000200020002", func(req *http.Request) (int, string) {
		return 200, "publickeys/unparseable"
	})

	kp, failure := keypairs.FetchPublicKey(suite.secretsClient, &apiModels.User{
		UserID: strfmt.UUID("00020002-0002-0002-0002-000200020002"),
	})
	suite.Nil(kp)
	suite.True(failure.Type.Matches(keypairs.FailPublicKey))
}

func (suite *KeypairFetcherTestSuite) TestFetchPublicKey_Success() {
	httpmock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 200)
	kp, failure := keypairs.FetchPublicKey(suite.secretsClient, &apiModels.User{
		UserID: strfmt.UUID("00020002-0002-0002-0002-000200020002"),
	})
	suite.Require().Nil(failure)
	suite.IsType(&keypairs.RSAPublicKey{}, kp)
}

func Test_KeypairCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairFetcherTestSuite))
}
