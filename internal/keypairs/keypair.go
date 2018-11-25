package keypairs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// MinimumRSABitLength is the minimum allowed bit-length when generating RSA keys.
const MinimumRSABitLength int = 12

var (
	// ErrBitLengthTooShort reflects an error when a key generation bit-length argument is too short.
	ErrBitLengthTooShort = errors.New("bit-length too short")

	// ErrInvalidPEMEncoding reflects an error trying to decode a PEM-encoded key.
	ErrInvalidPEMEncoding = errors.New("invalid PEM encoding")

	// ErrInvalidRSAPublicKey reflects an error trying to map a key to an RSAPublicKey.
	ErrInvalidRSAPublicKey = errors.New("key not an RSA public-key")
)

// Encrypter expects to encrypt a message.
type Encrypter interface {
	// Encrypt will encrypt the provided message using the Keypair's public-key.
	Encrypt(msg []byte) ([]byte, error)
}

// Decrypter expects to Decrypt some ciphertext.
type Decrypter interface {
	// Decrypt will decrypt the provided ciphertext using the Keypair's private-key.
	Decrypt(ciphertext []byte) ([]byte, error)
}

// Keypair provides behavior for working with public crypto key-pairs.
type Keypair interface {
	Encrypter
	Decrypter

	// EncodePrivateKey encodes the private-key for this key-pair to a human readable string.
	// Generally this will be encoded in some PEM format.
	EncodePrivateKey() string

	// EncodePublicKey encodes the public-key for this key-pair to a human readable string.
	// Generally this will be encoded in some PEM format.
	EncodePublicKey() (string, error)
}

// RSAKeypair implements a Keypair around an RSA private-key.
type RSAKeypair struct {
	*rsa.PrivateKey
}

// EncodePrivateKey will encode this RSA private-key to a PEM string.
func (keypair *RSAKeypair) EncodePrivateKey() string {
	keyBytes := x509.MarshalPKCS1PrivateKey(keypair.PrivateKey)
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		},
	)
	return string(keyPEM)
}

// EncodePublicKey will encode this RSA public-key to a PEM string.
func (keypair *RSAKeypair) EncodePublicKey() (string, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(&keypair.PublicKey)
	if err != nil {
		return "", err
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
	return rsaEncrypt(&keypair.PublicKey, msg)
}

// Decrypt will decrypt the provided ciphertext using the Keypair's private-key. This particular
// function will use SHA256 for the random oracle.
func (keypair *RSAKeypair) Decrypt(ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, keypair.PrivateKey, ciphertext, nil)
}

// GenerateRSA will generate an RSAKeypair instance given a bit-length.
// The value for bits can be anything `>= MinimumRSABitLength`.
func GenerateRSA(bits int) (*RSAKeypair, error) {
	if bits < MinimumRSABitLength {
		return nil, ErrBitLengthTooShort
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return &RSAKeypair{privateKey}, nil
}

// ParseRSA will parse a PEM encoded RSAKeypair.
func ParseRSA(privateKeyPEM string) (*RSAKeypair, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, ErrInvalidPEMEncoding
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &RSAKeypair{privKey}, nil
}

// RSAPublicKey implements an Encrypter around an RSA public-key.
type RSAPublicKey struct {
	*rsa.PublicKey
}

// Encrypt will encrypt the provided message using the Keypair's public-key. This particular
// function will use SHA256 for the random oracle.
func (key *RSAPublicKey) Encrypt(msg []byte) ([]byte, error) {
	return rsaEncrypt(key.PublicKey, msg)
}

// ParseRSAPublicKey will parse a PEM encoded RSAPublicKey
func ParseRSAPublicKey(publicKeyPEM string) (*RSAPublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, ErrInvalidPEMEncoding
	}

	ifc, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey, ok := ifc.(*rsa.PublicKey)
	if !ok {
		return nil, ErrInvalidRSAPublicKey
	}
	return &RSAPublicKey{pubKey}, nil
}

func rsaEncrypt(pubKey *rsa.PublicKey, msg []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, msg, nil)
}
