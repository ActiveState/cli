package keypairs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// MinimumRSABitLength is the minimum allowed bit-length when generating RSA keys.
const MinimumRSABitLength int = 12

// ErrBitLengthTooShort reflects an error when a key generation bit-length argument is too short.
var ErrBitLengthTooShort = errors.New("bit-length too short")

// Keypair provides behavior for working with public crypto key-pairs.
type Keypair interface {
	// EncodePrivateKey encodes the private-key for this key-pair to a human readable string.
	// Generally this will be encoding in some PEM format.
	EncodePrivateKey() string

	// EncodePublicKey encodes the public-key for this key-pair to a human readable string.
	// Generally this will be encoding in some PEM format.
	EncodePublicKey() (string, error)
}

// GeneratorFunc is a function that can generate new Keypairs.
type GeneratorFunc func() (Keypair, error)

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

// NewRSAGeneratorFunc will return a GeneratorFunc which can generate RSAKeypair instances.
// The value for bits can be anything `>= MinimumRSABitLength`.
func NewRSAGeneratorFunc(bits int) (GeneratorFunc, error) {
	if bits < MinimumRSABitLength {
		return nil, ErrBitLengthTooShort
	}

	return func() (Keypair, error) {
		privateKey, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return nil, err
		}
		return &RSAKeypair{privateKey}, nil
	}, nil
}
