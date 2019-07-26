package keypairs

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/keys"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailCrypto indicates a failure with something crypto related.
	FailCrypto = failures.Type("keypairs.fail.crypto")

	// FailKeypair represents a failure to successfully work with a Keypair.
	FailKeypair = failures.Type("keypairs.fail.keypair", FailCrypto)

	// FailKeypairParse indicates a failure to parse a keypair.
	FailKeypairParse = failures.Type("keypairs.fail.keypair.parse", FailKeypair)

	// FailKeypairPassphrase indicates a failure with passphrase.
	FailKeypairPassphrase = failures.Type("keypairs.fail.keypair.passphrase", FailKeypairParse, failures.FailUser)

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

	// EncodePublicKey encodes the public-key for this keypair to a human readable string.
	// Generally this will be encoded in some PEM format.
	EncodePublicKey() (string, *failures.Failure)

	// MatchPublicKey determines if a provided public-key in PEM encoded format matches this Keypair's
	// public-key.
	MatchPublicKey(publicKeyPEM string) bool
}

// EncodedKeypair encapulates a Keypair instance and the base-64 encodings on the public and private key
// components of that Keypair. This is especially useful in the event the private-key has been encrypted
// with a passphrase.
type EncodedKeypair struct {
	Keypair           Keypair
	EncodedPrivateKey string
	EncodedPublicKey  string
}

// EncodeKeypair returns an EncodedKeypair using the provided Keypair and secures the private-key with a
// passphrase.
func EncodeKeypair(keypair Keypair, passphrase string) (*EncodedKeypair, *failures.Failure) {
	var encodedPrivateKey string
	var failure *failures.Failure

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

// GenerateEncodedKeypair generates a new RSAKeypair, encrypts the private-key if a passphrase is provided,
// encodes the private and public keys, and returns they Keypair and encoded keys as an EncodedKeypair.
func GenerateEncodedKeypair(passphrase string, bits int) (*EncodedKeypair, *failures.Failure) {
	keypair, failure := GenerateRSA(bits)
	if failure != nil {
		return nil, failure
	}
	return EncodeKeypair(keypair, passphrase)
}

// SaveEncodedKeypair stores an encoded Keypair back to the Secrets Service.
func SaveEncodedKeypair(secretsClient *secretsapi.Client, encKeypair *EncodedKeypair) *failures.Failure {
	params := keys.NewSaveKeypairParams().WithKeypair(&secretsModels.KeypairChange{
		EncryptedPrivateKey: &encKeypair.EncodedPrivateKey,
		PublicKey:           &encKeypair.EncodedPublicKey,
	})

	if _, err := secretsClient.Keys.SaveKeypair(params, authentication.Get().ClientAuth()); err != nil {
		logging.Error("Saving keypair failed with error: %v", err)
		return secretsapi.FailKeypairSave.New("keypair_err_save")
	}

	// save the keypair locally to avoid authenticating the keypair every time it's used
	return SaveWithDefaults(encKeypair.Keypair)
}

// GenerateAndSaveEncodedKeypair first Generates and then tries to Save an EncodedKeypair. This is equivalent to calling
// GenerateEncodedKeypair and then SaveEncodedKeypair. Upon success of both actions, the EncodedKeypair will be returned,
// otherwise a Failure is returned.
func GenerateAndSaveEncodedKeypair(secretsClient *secretsapi.Client, passphrase string, bits int) (*EncodedKeypair, *failures.Failure) {
	encodedKeypair, failure := GenerateEncodedKeypair(passphrase, bits)
	if failure == nil {
		failure = SaveEncodedKeypair(secretsClient, encodedKeypair)
	}

	if failure != nil {
		return nil, failure
	}
	return encodedKeypair, nil
}
