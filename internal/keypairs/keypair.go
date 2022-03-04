package keypairs

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/keys"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// Encrypter expects to encrypt a message.
type Encrypter interface {
	// Encrypt will encrypt the provided message using the Keypair's public-key.
	Encrypt(msg []byte) ([]byte, error)

	// EncryptAndEncode will encrypt the provided message then it will base64 encode that ciphertext.
	EncryptAndEncode(msg []byte) (string, error)
}

// Decrypter expects to Decrypt some ciphertext.
type Decrypter interface {
	// Decrypt will decrypt the provided ciphertext using the Keypair's private-key.
	Decrypt(ciphertext []byte) ([]byte, error)

	// DecodeAndDecrypt will first base64 decode the provided msg then it will decrypt the resulting ciphertext.
	DecodeAndDecrypt(value string) ([]byte, error)
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
	EncryptAndEncodePrivateKey(passphrase string) (string, error)

	// EncodePublicKey encodes the public-key for this keypair to a human readable string.
	// Generally this will be encoded in some PEM format.
	EncodePublicKey() (string, error)

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
func EncodeKeypair(keypair Keypair, passphrase string) (*EncodedKeypair, error) {
	var encodedPrivateKey string
	var err error

	if passphrase == "" {
		encodedPrivateKey = keypair.EncodePrivateKey()
	} else {
		encodedPrivateKey, err = keypair.EncryptAndEncodePrivateKey(passphrase)
		if err != nil {
			return nil, errs.Wrap(err, "Could not encrypt encoded private key")
		}
	}

	encodedPublicKey, err := keypair.EncodePublicKey()
	if err != nil {
		return nil, errs.Wrap(err, "Could not encode public key")
	}

	return &EncodedKeypair{
		Keypair:           keypair,
		EncodedPrivateKey: encodedPrivateKey,
		EncodedPublicKey:  encodedPublicKey,
	}, nil
}

// GenerateEncodedKeypair generates a new RSAKeypair, encrypts the private-key if a passphrase is provided,
// encodes the private and public keys, and returns they Keypair and encoded keys as an EncodedKeypair.
func GenerateEncodedKeypair(passphrase string, bits int) (*EncodedKeypair, error) {
	keypair, err := GenerateRSA(bits)
	if err != nil {
		return nil, errs.Wrap(err, "Could not generate RSA")
	}
	return EncodeKeypair(keypair, passphrase)
}

// SaveEncodedKeypair stores an encoded Keypair back to the Secrets Service.
func SaveEncodedKeypair(cfg Configurable, secretsClient *secretsapi.Client, encKeypair *EncodedKeypair) error {
	params := keys.NewSaveKeypairParams().WithKeypair(&secretsModels.KeypairChange{
		EncryptedPrivateKey: &encKeypair.EncodedPrivateKey,
		PublicKey:           &encKeypair.EncodedPublicKey,
	})

	if _, err := secretsClient.Keys.SaveKeypair(params, authentication.LegacyGet().ClientAuth()); err != nil {
		return locale.WrapError(err, "keypair_err_save")
	}

	// save the keypair locally to avoid authenticating the keypair every time it's used
	return SaveWithDefaults(cfg, encKeypair.Keypair)
}

// GenerateAndSaveEncodedKeypair first Generates and then tries to Save an EncodedKeypair. This is equivalent to calling
// GenerateEncodedKeypair and then SaveEncodedKeypair. Upon success of both actions, the EncodedKeypair will be returned,
// otherwise a Failure is returned.
func GenerateAndSaveEncodedKeypair(cfg Configurable, secretsClient *secretsapi.Client, passphrase string, bits int) (*EncodedKeypair, error) {
	encodedKeypair, err := GenerateEncodedKeypair(passphrase, bits)
	if err != nil {
		return nil, errs.Wrap(err, "Could not generate encoded keypair")
	}

	err = SaveEncodedKeypair(cfg, secretsClient, encodedKeypair)
	if err != nil {
		return nil, errs.Wrap(err, "Could not save encoded keypair")
	}
	return encodedKeypair, nil
}
