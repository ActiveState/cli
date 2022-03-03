package keypairs_test

import (
	"net/http"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	apiModels "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
)

type KeypairFetcherTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	cfg           keypairs.Configurable
}

func (suite *KeypairFetcherTestSuite) BeforeTest(suiteName, testName string) {
	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient
	var err error
	suite.cfg, err = config.New()
	suite.NoError(err)

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairFetcherTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairFetcherTestSuite) TestFetch_NotFound() {
	httpmock.RegisterWithCode("GET", "/keypair", 404)
	kp, err := keypairs.Fetch(suite.secretsClient, suite.cfg, "")
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairFetcherTestSuite) TestFetch_ErrorParsing() {
	httpmock.RegisterWithResponder("GET", "/keypair", func(req *http.Request) (int, string) {
		return 200, "keypair-unparseable"
	})

	kp, err := keypairs.Fetch(suite.secretsClient, suite.cfg, "")
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairFetcherTestSuite) TestFetch_Success() {
	httpmock.RegisterWithCode("GET", "/keypair", 200)
	kp, err := keypairs.Fetch(suite.secretsClient, suite.cfg, "")
	suite.Require().Nil(err)
	suite.IsType(&keypairs.RSAKeypair{}, kp)
}

func (suite *KeypairFetcherTestSuite) TestFetchRaw_NotFound() {
	httpmock.RegisterWithCode("GET", "/keypair", 404)
	kp, err := keypairs.FetchRaw(suite.secretsClient, suite.cfg)
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairFetcherTestSuite) TestFetchRaw_Success() {
	httpmock.RegisterWithCode("GET", "/keypair", 200)
	kp, err := keypairs.FetchRaw(suite.secretsClient, suite.cfg)
	suite.Require().Nil(err)
	suite.IsType(&secrets_models.Keypair{}, kp)
}

func (suite *KeypairFetcherTestSuite) TestFetchPublicKey_NotFound() {
	httpmock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 404)
	kp, err := keypairs.FetchPublicKey(suite.secretsClient, &apiModels.User{
		UserID: strfmt.UUID("00020002-0002-0002-0002-000200020002"),
	})
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *KeypairFetcherTestSuite) TestFetchPublicKey_ErrorParsing() {
	httpmock.RegisterWithResponder("GET", "/publickeys/00020002-0002-0002-0002-000200020002", func(req *http.Request) (int, string) {
		return 200, "publickeys/unparseable"
	})

	key, err := keypairs.FetchPublicKey(suite.secretsClient, &apiModels.User{
		UserID: strfmt.UUID("00020002-0002-0002-0002-000200020002"),
	})
	suite.Nil(key)
	suite.Error(err)
}

func (suite *KeypairFetcherTestSuite) TestFetchPublicKey_Success() {
	httpmock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 200)
	kp, err := keypairs.FetchPublicKey(suite.secretsClient, &apiModels.User{
		UserID: strfmt.UUID("00020002-0002-0002-0002-000200020002"),
	})
	suite.Require().Nil(err)
	suite.IsType(&keypairs.RSAPublicKey{}, kp)
}

func Test_KeypairFetcher_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairFetcherTestSuite))
}
