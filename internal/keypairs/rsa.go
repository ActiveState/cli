package keypairs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

// MinimumRSABitLength is the minimum allowed bit-length when generating RSA keys.
const MinimumRSABitLength int = 1024

type ErrKeypairPassphrase struct{ *locale.LocalizedError }

// RSAKeypair implements a Keypair around an RSA private-key.
type RSAKeypair struct {
	*rsa.PrivateKey
}

// EncodePrivateKey will encode this RSA private-key to a PEM string.
func (keypair *RSAKeypair) EncodePrivateKey() string {
	return string(pem.EncodeToMemory(keypair.pemPrivateKeyBlock()))
}

// EncryptAndEncodePrivateKey will encrypt RSA private-key using the provided passphrase
// and then encode it to a PEM string.
func (keypair *RSAKeypair) EncryptAndEncodePrivateKey(passphrase string) (string, error) {
	block := keypair.pemPrivateKeyBlock()

	var err error
	block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(passphrase), x509.PEMCipherAES256)
	if err != nil {
		return "", errs.Wrap(err, "EncryptPEMBlock failed")
	}

	return string(pem.EncodeToMemory(block)), nil
}

func (keypair *RSAKeypair) pemPrivateKeyBlock() *pem.Block {
	return &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(keypair.PrivateKey),
	}
}

// EncodePublicKey will encode this RSA public-key to a PEM string.
func (keypair *RSAKeypair) EncodePublicKey() (string, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(&keypair.PublicKey)
	if err != nil {
		return "", errs.Wrap(err, "MarshalPKIXPublicKey failed")
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: keyBytes,
		},
	)
	return string(keyPEM), nil
}

// Encrypt will encrypt the provided message using the Keypair's public-key. This particular
// function will use SHA256 for the random oracle.
func (keypair *RSAKeypair) Encrypt(msg []byte) ([]byte, error) {
	b, err := rsaEncrypt(&keypair.PublicKey, msg)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// EncryptAndEncode will encrypt the provided message using the Keypair's public-key
// and then base-64 encode it.
func (keypair *RSAKeypair) EncryptAndEncode(msg []byte) (string, error) {
	encrBytes, err := keypair.Encrypt(msg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrBytes), nil
}

// Decrypt will decrypt the provided ciphertext using the Keypair's private-key. This particular
// function will use SHA256 for the random oracle.
func (keypair *RSAKeypair) Decrypt(ciphertext []byte) ([]byte, error) {
	b, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, keypair.PrivateKey, ciphertext, nil)
	if err != nil {
		return nil, errs.Wrap(err, "DecryptOAEP failed")
	}
	return b, nil
}

// DecodeAndDecrypt will base-64 decode the provided msg then decrypt the resulting ciphertext
// using the Keypair's private-key.
func (keypair *RSAKeypair) DecodeAndDecrypt(msg string) ([]byte, error) {
	encrBytes, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, locale.WrapError(err, "keypairs_err_base64_decoding")
	}
	return keypair.Decrypt(encrBytes)
}

// MatchPublicKey determines if a provided public-key in PEM encoded format matches this Keypair's
// public-key.
func (keypair *RSAKeypair) MatchPublicKey(publicKeyPEM string) bool {
	otherPublicKey, err := ParseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return false
	}

	return keypair.N.Cmp(otherPublicKey.N) == 0 && keypair.E == otherPublicKey.E
}

// GenerateRSA will generate an RSAKeypair instance given a bit-length.
// The value for bits can be anything `>= MinimumRSABitLength`.
func GenerateRSA(bits int) (*RSAKeypair, error) {
	if bits < MinimumRSABitLength {
		return nil, locale.NewError("keypairs_err_bitlength_too_short")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, errs.Wrap(err, "GenerateKey failed")
	}
	return &RSAKeypair{privateKey}, nil
}

// ParseRSA will parse a PEM encoded RSAKeypair. If the keypair is encrypted, it is expected that the
// passphrase is empty.
func ParseRSA(privateKeyPEM string) (*RSAKeypair, error) {
	return ParseEncryptedRSA(privateKeyPEM, "")
}

// ParseEncryptedRSA will parse a PEM encoded RSAKeypair that is possibly encrypted with a passphrase.
// If the keypair is not encrypted, the parsing will proceed uninterrupted.
func ParseEncryptedRSA(privateKeyPEM, passphrase string) (*RSAKeypair, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, locale.NewError("keypairs_err_pem_encoding")
	}

	var err error
	keyBytes := block.Bytes

	if x509.IsEncryptedPEMBlock(block) {
		keyBytes, err = x509.DecryptPEMBlock(block, []byte(passphrase))
		if err != nil {
			if err == x509.IncorrectPasswordError {
				return nil, &ErrKeypairPassphrase{locale.WrapError(err, "keypairs_err_passphrase_incorrect")}
			}
			return nil, errs.Wrap(err, "DecryptPEMBlock failed")
		}
	}

	privKey, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return nil, errs.Wrap(err, "ParsePKCS1PrivateKey failed")
	}
	return &RSAKeypair{privKey}, nil
}

// RSAPublicKey implements an Encrypter around an RSA public-key.
type RSAPublicKey struct {
	*rsa.PublicKey
}

// Encrypt will encrypt the provided message using this PublicKey. This particular
// function will use SHA256 for the random oracle.
func (key *RSAPublicKey) Encrypt(msg []byte) ([]byte, error) {
	b, err := rsaEncrypt(key.PublicKey, msg)
	if err != nil {
		return nil, errs.Wrap(err, "rsaEncrypt failed")
	}
	return b, nil
}

// EncryptAndEncode will encrypt the provided message using this PublicKey and then base-64 encode it.
func (key *RSAPublicKey) EncryptAndEncode(msg []byte) (string, error) {
	encrBytes, err := key.Encrypt(msg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrBytes), nil
}

// ParseRSAPublicKey will parse a PEM encoded RSAPublicKey
func ParseRSAPublicKey(publicKeyPEM string) (*RSAPublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, locale.NewError("keypairs_err_pem_encoding")
	}

	ifc, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errs.Wrap(err, "ParsePKIXPublicKey failed")
	}

	pubKey, ok := ifc.(*rsa.PublicKey)
	if !ok {
		return nil, locale.NewError("keypairs_err_invalid_rsa_publickey")
	}
	return &RSAPublicKey{pubKey}, nil
}

func rsaEncrypt(pubKey *rsa.PublicKey, msg []byte) ([]byte, error) {
	encrBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, msg, nil)
	if err != nil {
		return nil, errs.Wrap(err, "EncryptOAEP failed")
	}
	return encrBytes, nil
}
