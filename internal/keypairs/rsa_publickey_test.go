package keypairs_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"

	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/stretchr/testify/suite"
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
	suite.Require().NoError(err)

	decryptedBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, ciphertext, nil)
	suite.Require().NoError(err)
	suite.Equal("this is the catch", string(decryptedBytes))
}

func (suite *RSAPublicKeyTestSuite) TestParsePublicKey() {
	kp, err := keypairs.GenerateRSA(1024)
	suite.Require().NoError(err)
	pubKeyPEM, err := kp.EncodePublicKey()
	suite.Require().NoError(err)

	pubKey, err := keypairs.ParseRSAPublicKey(pubKeyPEM)
	suite.Require().NoError(err)

	suite.Equal(kp.PublicKey, *pubKey.PublicKey)
}

func Test_RSAPublicKey_TestSuite(t *testing.T) {
	suite.Run(t, new(RSAPublicKeyTestSuite))
}
