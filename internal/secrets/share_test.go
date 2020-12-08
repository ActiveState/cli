package secrets_test

import (
	"testing"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/secrets"
	secrets_models "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type SecretsSharingTestSuite struct {
	suite.Suite

	sourceKeypair keypairs.Keypair

	targetKeypair keypairs.Keypair
	targetPubKey  string
}

func (suite *SecretsSharingTestSuite) SetupSuite() {
	var failure error

	suite.sourceKeypair, failure = keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)

	suite.targetKeypair, failure = keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)

	suite.targetPubKey, failure = suite.targetKeypair.EncodePublicKey()
	suite.Require().Nil(failure)
}

func (suite *SecretsSharingTestSuite) TestFailure_ParsingPublicKey() {
	badPubKey := "-- BEGIN BAD RSA PUB KEY --\nabc123\n-- END BAD RSA PUB KEY --"
	newShares, failure := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &badPubKey,
	})
	suite.Nil(newShares)
	suite.Equal(keypairs.FailPublicKeyParse, failure.Type, "failure type mismatch")

}

func (suite *SecretsSharingTestSuite) TestFailure_FirstShareHasBadlyEncryptedValue() {
	newShares, failure := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &suite.targetPubKey,
		Shares:    []*secrets_models.UserSecretShare{newUserSecretShare("", "FOO", "badly encrypted value")},
	})
	suite.Nil(newShares)
	suite.Equal(keypairs.FailKeyDecode, failure.Type, "failure type mismatch")
}

func (suite *SecretsSharingTestSuite) TestFailure_FailedToEncryptForTargetUser() {
	shortKeypair, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	// this is a valid public key, but will be too short for encrypting with
	shortPubKey, failure := shortKeypair.EncodePublicKey()
	suite.Require().Nil(failure)

	encrValue, failure := suite.sourceKeypair.EncryptAndEncode([]byte("luv 2 encrypt data"))
	suite.Require().Nil(failure)

	newShares, failure := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &shortPubKey,
		Shares:    []*secrets_models.UserSecretShare{newUserSecretShare("", "FOO", encrValue)},
	})
	suite.Nil(newShares)
	suite.Equal(keypairs.FailPublicKey, failure.Type, "failure type mismatch")
}

func (suite *SecretsSharingTestSuite) TestSuccess_ReceivedEmptySharesList() {
	newShares, failure := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &suite.targetPubKey,
		Shares:    []*secrets_models.UserSecretShare{},
	})
	suite.Len(newShares, 0)
	suite.Nil(failure)
}

func (suite *SecretsSharingTestSuite) TestSuccess_MultipleSharesProcessed() {
	encrOrgSecret, failure := suite.sourceKeypair.EncryptAndEncode([]byte("org secret"))
	suite.Require().Nil(failure)

	projID := strfmt.UUID("00020002-0002-0002-0002-000200020002")
	encrProjSecret, failure := suite.sourceKeypair.EncryptAndEncode([]byte("proj secret"))
	suite.Require().Nil(failure)

	newShares, failure := secrets.ShareFromDiff(suite.sourceKeypair, &secrets_models.UserSecretDiff{
		PublicKey: &suite.targetPubKey,
		Shares: []*secrets_models.UserSecretShare{
			newUserSecretShare("", "org-secret", encrOrgSecret),
			newUserSecretShare(projID, "proj-secret", encrProjSecret),
		},
	})

	suite.Require().Nil(failure)
	suite.Require().Len(newShares, 2)

	decrOrgSecret, failure := suite.targetKeypair.DecodeAndDecrypt(*newShares[0].Value)
	suite.Require().Nil(failure)
	suite.Equal("org secret", string(decrOrgSecret))
	suite.Equal("org-secret", *newShares[0].Name)
	suite.Zero(newShares[0].ProjectID)

	decrProjSecret, failure := suite.targetKeypair.DecodeAndDecrypt(*newShares[1].Value)
	suite.Require().Nil(failure)
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
