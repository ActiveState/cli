package secrets_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
)

type SecretsSharingTestSuite struct {
	suite.Suite

	sourceKeypair keypairs.Keypair

	targetKeypair keypairs.Keypair
	targetPubKey  string
}

func (suite *SecretsSharingTestSuite) SetupSuite() {
	var err error

	suite.sourceKeypair, err = keypairs.GenerateRSA(2048)
	suite.Require().NoError(err)

	suite.targetKeypair, err = keypairs.GenerateRSA(2048)
	suite.Require().NoError(err)

	suite.targetPubKey, err = suite.targetKeypair.EncodePublicKey()
	suite.Require().NoError(err)
}

func (suite *SecretsSharingTestSuite) TestFailure_ParsingPublicKey() {
	badPubKey := "-- BEGIN BAD RSA PUB KEY --\nabc123\n-- END BAD RSA PUB KEY --"
	newShares, err := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &badPubKey,
	})
	suite.Nil(newShares)
	suite.Require().Error(err)

}

func (suite *SecretsSharingTestSuite) TestFailure_FirstShareHasBadlyEncryptedValue() {
	newShares, err := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &suite.targetPubKey,
		Shares:    []*secrets_models.UserSecretShare{newUserSecretShare("", "FOO", "badly encrypted value")},
	})
	suite.Nil(newShares)
	suite.Require().Error(err)
}

func (suite *SecretsSharingTestSuite) TestSuccess_ReceivedEmptySharesList() {
	newShares, err := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &suite.targetPubKey,
		Shares:    []*secrets_models.UserSecretShare{},
	})
	suite.Len(newShares, 0)
	suite.Nil(err)
}

func (suite *SecretsSharingTestSuite) TestSuccess_MultipleSharesProcessed() {
	encrOrgSecret, err := suite.sourceKeypair.EncryptAndEncode([]byte("org secret"))
	suite.Require().NoError(err)

	projID := strfmt.UUID("00020002-0002-0002-0002-000200020002")
	encrProjSecret, err := suite.sourceKeypair.EncryptAndEncode([]byte("proj secret"))
	suite.Require().NoError(err)

	newShares, err := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &suite.targetPubKey,
		Shares: []*secrets_models.UserSecretShare{
			newUserSecretShare("", "org-secret", encrOrgSecret),
			newUserSecretShare(projID, "proj-secret", encrProjSecret),
		},
	})

	suite.Require().NoError(err)
	suite.Require().Len(newShares, 2)

	decrOrgSecret, err := suite.targetKeypair.DecodeAndDecrypt(*newShares[0].Value)
	suite.Require().NoError(err)
	suite.Equal("org secret", string(decrOrgSecret))
	suite.Equal("org-secret", *newShares[0].Name)
	suite.Zero(newShares[0].ProjectID)

	decrProjSecret, err := suite.targetKeypair.DecodeAndDecrypt(*newShares[1].Value)
	suite.Require().NoError(err)
	suite.Equal("proj secret", string(decrProjSecret))
	suite.Equal("proj-secret", *newShares[1].Name)
	suite.Equal(projID, newShares[1].ProjectID)
}

func newUserSecretShare(projID strfmt.UUID, name, encrValue string) *secrets_models.UserSecretShare {
	return &secrets_models.UserSecretShare{
		ProjectID: projID,
		Name:      &name,
		Value:     &encrValue,
	}
}

func Test_SecretsSharing_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsSharingTestSuite))
}
