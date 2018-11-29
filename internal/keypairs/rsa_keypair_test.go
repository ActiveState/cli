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
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength - 1)
	suite.Nil(kp)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairGenerate), "Did not expect failure type: %s", failure.Type.Name)
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_UsesMinimumBitLength() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)
	suite.NotNil(kp)
	suite.Implements((*keypairs.Keypair)(nil), kp)
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_GeneratesRSAKeypair() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)
	suite.Regexp(`^-{5}BEGIN RSA PRIVATE KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PRIVATE KEY-{5}\s`, kp.EncodePrivateKey())

	encPubKey, failure := kp.EncodePublicKey()
	suite.Require().Nil(failure)
	suite.Regexp(`^-{5}BEGIN RSA PUBLIC KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PUBLIC KEY-{5}\s`, encPubKey)
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_MessageTooLongForKeySize() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	encMsg, failure := kp.Encrypt([]byte("howdy doody"))
	suite.Nil(encMsg)
	suite.Truef(failure.Type.Matches(keypairs.FailEncrypt), "Did not expect failure type: %s", failure.Type.Name)
	suite.Contains(failure.Error(), "message too long")
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncryptsAndDecrypts() {
	kp, failure := keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)

	encryptedMsg, failure := kp.Encrypt([]byte("howdy doody"))
	suite.Require().Nil(failure)
	suite.NotEqual("howdy doody", string(encryptedMsg))

	decryptedMsg, failure := kp.Decrypt(encryptedMsg)
	suite.Require().Nil(failure)
	suite.Equal("howdy doody", string(decryptedMsg))
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncodesAndDeccodesEncryptedValues() {
	kp, failure := keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)

	encryptedMsg, failure := kp.EncryptAndEncode([]byte("howdy doody"))
	suite.Require().Nil(failure)
	suite.NotEqual("howdy doody", encryptedMsg)

	decryptedMsg, failure := kp.DecodeAndDecrypt(encryptedMsg)
	suite.Require().Nil(failure)
	suite.Equal("howdy doody", string(decryptedMsg))
}

func (suite *RSAKeypairTestSuite) TestParseRSA_ParsesKeypair() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	encPrivKey := kp.EncodePrivateKey()
	kp2, failure := keypairs.ParseRSA(encPrivKey)
	suite.Require().Nil(failure)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_EncodingNotOfPrivateKey() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	encPublicKey, failure := kp.EncodePublicKey()
	suite.Require().Nil(failure)

	kp2, failure := keypairs.ParseRSA(encPublicKey)
	suite.Nil(kp2)
	suite.Contains(failure.Error(), "structure error")
}

func (suite *RSAKeypairTestSuite) TestParseRSA_KeypairNotPEMEncoded() {
	kp, failure := keypairs.ParseRSA("this is not an encoded key")
	suite.Nil(kp)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairParse), "Did not expect failure type: %s", failure.Type.Name)
}

func Test_RSAKeypair_TestSuite(t *testing.T) {
	suite.Run(t, new(RSAKeypairTestSuite))
}
