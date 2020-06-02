package keypairs_test

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
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

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncryptAndEncodePrivateKey() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	keyPEM, failure := kp.EncryptAndEncodePrivateKey("abc123")
	suite.Require().Nil(failure)
	suite.Regexp(`^-{5}BEGIN RSA PRIVATE KEY-{5}`, keyPEM)
	suite.Regexp(`Proc-Type:\s+[\d+],ENCRYPTED`, keyPEM)
	suite.Regexp(`DEK-Info:\s+AES-256-CBC`, keyPEM)
	suite.Regexp(`[[:alnum:]/+=]{44}`, keyPEM)
	suite.Regexp(`-{5}END RSA PRIVATE KEY-{5}\s`, keyPEM)
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

	encPubKey, failure := kp.EncodePublicKey()
	suite.Require().Nil(failure)

	kp2, failure := keypairs.ParseRSA(encPubKey)
	suite.Nil(kp2)
	suite.True(failure.Type.Matches(keypairs.FailKeypairParse))
	suite.Contains(failure.Error(), "structure error")
}

func (suite *RSAKeypairTestSuite) TestParseRSA_RequiresPassphrase() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	keyBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	newKeyBlock, err := x509.EncryptPEMBlock(rand.Reader, keyBlock.Type, keyBlock.Bytes, []byte("abc123"), x509.PEMCipherAES256)
	suite.Require().NoError(err)
	encrPrivKey := string(pem.EncodeToMemory(newKeyBlock))

	kp2, failure := keypairs.ParseRSA(encrPrivKey)
	suite.Nil(kp2)
	suite.True(failure.Type.Matches(keypairs.FailKeypairPassphrase), "failure was: %v", failure)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_EmptyPassphraseButStillEncrypted() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	keyBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	newKeyBlock, err := x509.EncryptPEMBlock(rand.Reader, keyBlock.Type, keyBlock.Bytes, []byte(""), x509.PEMCipherAES256)
	suite.Require().NoError(err)
	encrPrivKey := string(pem.EncodeToMemory(newKeyBlock))

	kp2, failure := keypairs.ParseRSA(encrPrivKey)
	suite.Require().Nil(failure)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_KeypairNotPEMEncoded() {
	kp, failure := keypairs.ParseRSA("this is not an encoded key")
	suite.Nil(kp)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairParse), "Did not expect failure type: %s", failure.Type.Name)
}

func (suite *RSAKeypairTestSuite) TestParseEncryptedRSA_ParsesKeypair() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	keyPEM, failure := kp.EncryptAndEncodePrivateKey("abc123")
	suite.Require().Nil(failure)

	kp2, failure := keypairs.ParseEncryptedRSA(keyPEM, "abc123")
	suite.Require().Nil(failure)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseEncryptedRSA_KeypairNotPEMEncoded() {
	kp, failure := keypairs.ParseEncryptedRSA("this is not an encoded key", "")
	suite.Nil(kp)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairParse), "Did not expect failure type: %s", failure.Type.Name)
}

func (suite *RSAKeypairTestSuite) TestParseEncryptedRSA_IncorrectPassphrase() {
	kp, failure := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(failure)

	keyPEM, failure := kp.EncryptAndEncodePrivateKey("abc123")
	suite.Require().Nil(failure)

	kp2, failure := keypairs.ParseEncryptedRSA(keyPEM, "notTheRightPassphrase")
	suite.Require().Nil(kp2)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairPassphrase), "Did not expect failure type: %s", failure.Type.Name)
}

func (suite *RSAKeypairTestSuite) TestMatchPublicKey_DifferentPublicKeys() {
	kp1, failure := keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)
	kp2, failure := keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)

	pubkey2PEM, failure := kp2.EncodePublicKey()
	suite.Require().Nil(failure)

	suite.False(kp1.MatchPublicKey(pubkey2PEM))
}

func (suite *RSAKeypairTestSuite) TestMatchPublicKey_SamePublicKey() {
	kp, failure := keypairs.GenerateRSA(1024)
	suite.Require().Nil(failure)

	pubkeyPEM, failure := kp.EncodePublicKey()
	suite.Require().Nil(failure)

	suite.True(kp.MatchPublicKey(pubkeyPEM))
}

func Test_RSAKeypair_TestSuite(t *testing.T) {
	suite.Run(t, new(RSAKeypairTestSuite))
}
