package keypairs_test

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/keypairs"
)

type RSAKeypairTestSuite struct {
	suite.Suite
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_ErrorBitLengthLessThanMin() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength - 1)
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_UsesMinimumBitLength() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)
	suite.NotNil(kp)
	suite.Implements((*keypairs.Keypair)(nil), kp)
}

func (suite *RSAKeypairTestSuite) TestGenerateRSA_GeneratesRSAKeypair() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)
	suite.Regexp(`^-{5}BEGIN RSA PRIVATE KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PRIVATE KEY-{5}\s`, kp.EncodePrivateKey())

	encPubKey, err := kp.EncodePublicKey()
	suite.Require().Nil(err)
	suite.Regexp(`^-{5}BEGIN RSA PUBLIC KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PUBLIC KEY-{5}\s`, encPubKey)
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncryptAndEncodePrivateKey() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	keyPEM, err := kp.EncryptAndEncodePrivateKey("abc123")
	suite.Require().Nil(err)
	suite.Regexp(`^-{5}BEGIN RSA PRIVATE KEY-{5}`, keyPEM)
	suite.Regexp(`Proc-Type:\s+[\d+],ENCRYPTED`, keyPEM)
	suite.Regexp(`DEK-Info:\s+AES-256-CBC`, keyPEM)
	suite.Regexp(`[[:alnum:]/+=]{44}`, keyPEM)
	suite.Regexp(`-{5}END RSA PRIVATE KEY-{5}\s`, keyPEM)
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_MessageTooLongForKeySize() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	encMsg, err := kp.Encrypt([]byte("howdy doody"))
	suite.Nil(encMsg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "EncryptOAEP failed")
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncryptsAndDecrypts() {
	kp, err := keypairs.GenerateRSA(1024)
	suite.Require().Nil(err)

	encryptedMsg, err := kp.Encrypt([]byte("howdy doody"))
	suite.Require().Nil(err)
	suite.NotEqual("howdy doody", string(encryptedMsg))

	decryptedMsg, err := kp.Decrypt(encryptedMsg)
	suite.Require().Nil(err)
	suite.Equal("howdy doody", string(decryptedMsg))
}

func (suite *RSAKeypairTestSuite) TestRSAKeypair_EncodesAndDeccodesEncryptedValues() {
	kp, err := keypairs.GenerateRSA(1024)
	suite.Require().Nil(err)

	encryptedMsg, err := kp.EncryptAndEncode([]byte("howdy doody"))
	suite.Require().Nil(err)
	suite.NotEqual("howdy doody", encryptedMsg)

	decryptedMsg, err := kp.DecodeAndDecrypt(encryptedMsg)
	suite.Require().Nil(err)
	suite.Equal("howdy doody", string(decryptedMsg))
}

func (suite *RSAKeypairTestSuite) TestParseRSA_ParsesKeypair() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	encPrivKey := kp.EncodePrivateKey()
	kp2, err := keypairs.ParseRSA(encPrivKey)
	suite.Require().Nil(err)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_EncodingNotOfPrivateKey() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	encPubKey, err := kp.EncodePublicKey()
	suite.Require().Nil(err)

	kp2, err := keypairs.ParseRSA(encPubKey)
	suite.Nil(kp2)
	suite.Error(err)
	suite.Contains(err.Error(), "ParsePKCS1PrivateKey failed")
}

func (suite *RSAKeypairTestSuite) TestParseRSA_RequiresPassphrase() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	keyBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	newKeyBlock, err := x509.EncryptPEMBlock(rand.Reader, keyBlock.Type, keyBlock.Bytes, []byte("abc123"), x509.PEMCipherAES256)
	suite.Require().NoError(err)
	encrPrivKey := string(pem.EncodeToMemory(newKeyBlock))

	kp2, err := keypairs.ParseRSA(encrPrivKey)
	suite.Nil(kp2)
	suite.Error(err)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_EmptyPassphraseButStillEncrypted() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	keyBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	newKeyBlock, err := x509.EncryptPEMBlock(rand.Reader, keyBlock.Type, keyBlock.Bytes, []byte(""), x509.PEMCipherAES256)
	suite.Require().NoError(err)
	encrPrivKey := string(pem.EncodeToMemory(newKeyBlock))

	kp2, err := keypairs.ParseRSA(encrPrivKey)
	suite.Require().Nil(err)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseRSA_KeypairNotPEMEncoded() {
	kp, err := keypairs.ParseRSA("this is not an encoded key")
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *RSAKeypairTestSuite) TestParseEncryptedRSA_ParsesKeypair() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	keyPEM, err := kp.EncryptAndEncodePrivateKey("abc123")
	suite.Require().Nil(err)

	kp2, err := keypairs.ParseEncryptedRSA(keyPEM, "abc123")
	suite.Require().Nil(err)
	suite.Implements((*keypairs.Keypair)(nil), kp2)
	suite.Equal(kp, kp2)
}

func (suite *RSAKeypairTestSuite) TestParseEncryptedRSA_KeypairNotPEMEncoded() {
	kp, err := keypairs.ParseEncryptedRSA("this is not an encoded key", "")
	suite.Nil(kp)
	suite.Error(err)
}

func (suite *RSAKeypairTestSuite) TestParseEncryptedRSA_IncorrectPassphrase() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)

	keyPEM, err := kp.EncryptAndEncodePrivateKey("abc123")
	suite.Require().Nil(err)

	kp2, err := keypairs.ParseEncryptedRSA(keyPEM, "notTheRightPassphrase")
	suite.Require().Nil(kp2)
	suite.Error(err)
}


func Test_RSAKeypair_TestSuite(t *testing.T) {
	suite.Run(t, new(RSAKeypairTestSuite))
}
