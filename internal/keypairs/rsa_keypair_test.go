package keypairs_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/stretchr/testify/suite"
)

type RSAKeypairTestSuite struct {
	suite.Suite
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_ErrorBitLengthLessThanMin() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength - 1)
	suite.Nil(kp)
	suite.Equal(err, keypairs.ErrBitLengthTooShort)
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_UsesMinimumBitLength() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().NoError(err)
	suite.NotNil(kp)
	suite.Implements((*keypairs.Keypair)(nil), kp)
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_GeneratesRSAKeypair() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().NoError(err)
	suite.Regexp(`^-{5}BEGIN RSA PRIVATE KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PRIVATE KEY-{5}\s`, kp.EncodePrivateKey())

	encPubKey, err := kp.EncodePublicKey()
	suite.Require().NoError(err, "encoding public key")
	suite.Regexp(`^-{5}BEGIN RSA PUBLIC KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PUBLIC KEY-{5}\s`, encPubKey)
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_MessageTooLongForKeySize() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().NoError(err)

	encMsg, err := kp.Encrypt([]byte("howdy doody"))
	suite.Nil(encMsg)
	suite.Contains(err.Error(), "message too long")
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncryptsAndDecrypts() {
	kp, err := keypairs.GenerateRSA(1024)
	suite.Require().NoError(err)

	encryptedMsg, err := kp.Encrypt([]byte("howdy doody"))
	suite.Require().NoError(err)
	suite.NotEqual("howdy doody", string(encryptedMsg))

	decryptedMsg, err := kp.Decrypt(encryptedMsg)
	suite.Require().NoError(err)
	suite.Equal("howdy doody", string(decryptedMsg))
}

func (suite *RSAKeypairTestSuite) TestParseRSA_ParsesKeypair() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().NoError(err)

	encPrivKey := kp.EncodePrivateKey()
	kp2, err := keypairs.ParseRSA(encPrivKey)
	suite.Require().NoError(err)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_EncodingNotOfPrivateKey() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().NoError(err)

	encPublicKey, err := kp.EncodePublicKey()
	suite.Require().NoError(err)

	kp2, err := keypairs.ParseRSA(encPublicKey)
	suite.Nil(kp2)
	suite.Contains(err.Error(), "structure error")
}

func (suite *RSAKeypairTestSuite) TestParseRSA_KeypairNotPEMEncoded() {
	kp, err := keypairs.ParseRSA("this is not an encoded key")
	suite.Nil(kp)
	suite.Require().Equal(err, keypairs.ErrInvalidPEMEncoding)
}

func Test_RSAKeypair_TestSuite(t *testing.T) {
	suite.Run(t, new(RSAKeypairTestSuite))
}
