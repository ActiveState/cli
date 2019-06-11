package secrets

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/failures"
)

func TestGetUserSecret(t *testing.T) {
	secret, fail := getSecret("user.foo")
	require.NoError(t, fail.ToError())
	require.True(t, secret.IsUser(), "Is user secret")
}

func TestGetProjectSecret(t *testing.T) {
	secret, fail := getSecret("project.foo")
	require.NoError(t, fail.ToError())
	require.True(t, secret.IsProject(), "Is project secret")
}

func TestGetSecretFailTooManyDots(t *testing.T) {
	_, fail := getSecret("project.toomanydots.foo")
	require.Error(t, fail.ToError())
	require.Equal(t, failures.FailUserInput.Name, fail.Type.Name)
}

func TestGetSecretFailScope(t *testing.T) {
	_, fail := getSecret("invalid.foo")
	require.Error(t, fail.ToError())
	require.Equal(t, failures.FailInput.Name, fail.Type.Name)
}
