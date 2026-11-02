package keypairs_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireSameRSAKey asserts that two keypairs hold the same RSA private key.
// Comparing the structs directly also compares rsa.PrivateKey's precomputed
// internals, whose representation can differ between a generated key and the
// same key parsed back from PEM.
func requireSameRSAKey(t *testing.T, expected, actual keypairs.Keypair) {
	t.Helper()
	expectedRSA, ok := expected.(*keypairs.RSAKeypair)
	require.True(t, ok, "expected keypair is not an RSA keypair")
	actualRSA, ok := actual.(*keypairs.RSAKeypair)
	require.True(t, ok, "actual keypair is not an RSA keypair")
	assert.True(t, expectedRSA.PrivateKey.Equal(actualRSA.PrivateKey), "RSA private keys differ")
}
