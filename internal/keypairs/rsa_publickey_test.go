package keypairs_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/keypairs"
)

type RSAPublicKeyTestSuite struct {
	suite.Suite
}

func (suite *RSAPublicKeyTestSuite) TestIsEncrypter() {
	suite.Implements((*keypairs.Encrypter)(nil), &keypairs.RSAPublicKey{})
}

func (suite *RSAPublicKeyTestSuite) TestEncrypts() {
	privKey, err := rsa.GenerateKey(rand.Reader, 1024)
	suite.Require().NoError(err)

	pubKey := &keypairs.RSAPublicKey{PublicKey: &privKey.PublicKey}
	ciphertext, err := pubKey.Encrypt([]byte("this is the catch"))
	suite.Require().Nil(err)

	decryptedBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, ciphertext, nil)
	suite.Require().NoError(err)
	suite.Equal("this is the catch", string(decryptedBytes))
}

func (suite *RSAPublicKeyTestSuite) TestEncryptsAndEncodes() {
	privKey, err := rsa.GenerateKey(rand.Reader, 1024)
	suite.Require().NoError(err)

	pubKey := &keypairs.RSAPublicKey{PublicKey: &privKey.PublicKey}
	encrEncodedStr, err := pubKey.EncryptAndEncode([]byte("this is the catch"))
	suite.Require().Nil(err)

	encrBytes, err := base64.StdEncoding.DecodeString(encrEncodedStr)
	suite.Require().NoError(err)

	decryptedBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, encrBytes, nil)
	suite.Require().NoError(err)
	suite.Equal("this is the catch", string(decryptedBytes))
}

func (suite *RSAPublicKeyTestSuite) TestParsePublicKey() {
	kp, err := keypairs.GenerateRSA(keypairs.MinimumRSABitLength)
	suite.Require().Nil(err)
	pubKeyPEM, err := kp.EncodePublicKey()
	suite.Require().Nil(err)

	pubKey, err := keypairs.ParseRSAPublicKey(pubKeyPEM)
	suite.Require().Nil(err)

	suite.Equal(kp.PublicKey, *pubKey.PublicKey)
}

func Test_RSAPublicKey_TestSuite(t *testing.T) {
	suite.Run(t, new(RSAPublicKeyTestSuite))
}
