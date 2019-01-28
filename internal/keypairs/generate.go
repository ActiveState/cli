package keypairs

import (
	"github.com/ActiveState/cli/internal/failures"
)

// EncodedKeypair encapulates a Keypair instance and the base-64 encodings on the public and private key
// components of that Keypair. This is especially useful in the event the private-key has been encrypted
// with a passphrase.
type EncodedKeypair struct {
	Keypair           Keypair
	EncodedPrivateKey string
	EncodedPublicKey  string
}

// GenerateEncodedKeypair generates a new RSAKeypair, encrypts the private-key if a passphrase is provided,
// encodes the private and public keys, and returns they Keypair and encoded keys as an EncodedKeypair.
func GenerateEncodedKeypair(passphrase string, bits int) (*EncodedKeypair, *failures.Failure) {
	keypair, failure := GenerateRSA(bits)
	if failure != nil {
		return nil, failure
	}

	var encodedPrivateKey string
	if passphrase == "" {
		encodedPrivateKey = keypair.EncodePrivateKey()
	} else {
		encodedPrivateKey, failure = keypair.EncryptAndEncodePrivateKey(passphrase)
		if failure != nil {
			return nil, failure
		}
	}

	encodedPublicKey, failure := keypair.EncodePublicKey()
	if failure != nil {
		return nil, failure
	}

	return &EncodedKeypair{
		Keypair:           keypair,
		EncodedPrivateKey: encodedPrivateKey,
		EncodedPublicKey:  encodedPublicKey,
	}, nil
}
