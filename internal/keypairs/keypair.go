package keypairs

import (
	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailCrypto indicates a failure with something crypto related.
	FailCrypto = failures.Type("keypairs.fail.crypto")

	// FailKeypair represents a failure to successfully work with a Keypair.
	FailKeypair = failures.Type("keypairs.fail.keypair", FailCrypto)

	// FailKeypairParse indicates a failure to parse a keypair.
	FailKeypairParse = failures.Type("keypairs.fail.keypair.parse", FailKeypair)

	// FailKeypairPassphrase indicates a failure with passphrase.
	FailKeypairPassphrase = failures.Type("keypairs.fail.keypair.passphrase", FailKeypairParse)

	// FailKeypairGenerate indicates a failure to generate a keypair.
	FailKeypairGenerate = failures.Type("keypairs.fail.keypair.generate", FailKeypair)

	// FailPublicKey represents a failure to successfully work with a PublicKey.
	FailPublicKey = failures.Type("keypairs.fail.publickey")

	// FailPublicKeyParse indicates a failure to parse a public-key.
	FailPublicKeyParse = failures.Type("keypairs.fail.publickey.parse", FailPublicKey)

	// FailKeyDecode indicates a failure to decode a key.
	FailKeyDecode = failures.Type("keypairs.fail.key.decode", FailCrypto)

	// FailKeyEncode indicates a failure to encode a key.
	FailKeyEncode = failures.Type("keypairs.fail.key.encode", FailCrypto)

	// FailDecrypt indicates a failure to decrypt a value.
	FailDecrypt = failures.Type("keypairs.fail.decrypt", FailCrypto)

	// FailEncrypt indicates a failure to decrypt a value.
	FailEncrypt = failures.Type("keypairs.fail.encrypt", FailCrypto)
)

// Encrypter expects to encrypt a message.
type Encrypter interface {
	// Encrypt will encrypt the provided message using the Keypair's public-key.
	Encrypt(msg []byte) ([]byte, *failures.Failure)

	// EncryptAndEncode will encrypt the provided message then it will base64 encode that ciphertext.
	EncryptAndEncode(msg []byte) (string, *failures.Failure)
}

// Decrypter expects to Decrypt some ciphertext.
type Decrypter interface {
	// Decrypt will decrypt the provided ciphertext using the Keypair's private-key.
	Decrypt(ciphertext []byte) ([]byte, *failures.Failure)

	// DecodeAndDecrypt will first base64 decode the provided msg then it will decrypt the resulting ciphertext.
	DecodeAndDecrypt(value string) ([]byte, *failures.Failure)
}

// Keypair provides behavior for working with public crypto key-pairs.
type Keypair interface {
	Encrypter
	Decrypter

	// EncodePrivateKey encodes the private-key for this key-pair to a human readable string.
	// Generally this will be encoded in some PEM format.
	EncodePrivateKey() string

	// EncryptAndEncodePrivateKey encodes the private-key for this key-pair to a human readable string.
	// Generally this will be encoded in some PEM format. First though, the private-key will be
	// encrypted using the provided passphrase.
	EncryptAndEncodePrivateKey(passphrase string) (string, *failures.Failure)

	// EncodePublicKey encodes the public-key for this key-pair to a human readable string.
	// Generally this will be encoded in some PEM format.
	EncodePublicKey() (string, *failures.Failure)
}
