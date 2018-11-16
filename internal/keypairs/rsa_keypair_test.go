package keypairs_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeypair_NewRSAGeneratorFunc_ErrorBitLengthLTMinAllowed(t *testing.T) {
	assert := assert.New(t)
	genFn, err := keypairs.NewRSAGeneratorFunc(keypairs.MinimumRSABitLength - 1)
	assert.Nil(genFn)
	assert.Equal(err, keypairs.ErrBitLengthTooShort)
}

func TestKeypair_NewRSAGeneratorFunc_UsesMinimumBitLength(t *testing.T) {
	assert := assert.New(t)
	genFn, err := keypairs.NewRSAGeneratorFunc(keypairs.MinimumRSABitLength)
	assert.NotNil(genFn)
	assert.NoError(err)
}

func TestKeypair_NewRSAGeneratorFunc_GeneratesRSAKeypairs(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	genFn, err := keypairs.NewRSAGeneratorFunc(keypairs.MinimumRSABitLength)
	require.NoError(err)

	kp, err := genFn()
	require.NoError(err, "creating generator function")
	assert.IsType(new(keypairs.RSAKeypair), kp, "generating new RSAKeypair")

	assert.Regexp(`^-{5}BEGIN RSA PRIVATE KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PRIVATE KEY-{5}\s`, kp.EncodePrivateKey())

	encPubKey, err := kp.EncodePublicKey()
	require.NoError(err, "encoding public key")
	assert.Regexp(`^-{5}BEGIN RSA PUBLIC KEY-{5}\s[[:alnum:]/+=]{44}\s-{5}END RSA PUBLIC KEY-{5}\s`, encPubKey)
}
