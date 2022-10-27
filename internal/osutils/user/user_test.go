package user

import (
	"os"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserHome(t *testing.T) {
	osHomeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	userHomeDir, err := HomeDir()
	require.NoError(t, err)
	assert.Equal(t, userHomeDir, osHomeDir)
}

func TestActiveStateHome(t *testing.T) {
	os.Setenv(constants.HomeEnvVarName, "override")
	defer func() { os.Unsetenv(constants.HomeEnvVarName) }()
	userHomeDir, err := HomeDir()
	require.NoError(t, err)
	assert.Equal(t, userHomeDir, "override")
}

func TestNoHome(t *testing.T) {
	osHomeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		os.Unsetenv("HOME")
		defer func() { os.Setenv("HOME", osHomeDir) }()
	} else {
		os.Unsetenv("USERPROFILE")
		defer func() { os.Setenv("USERPROFILE", osHomeDir) }()
	}
	_, err = HomeDir()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HOME environment variable is unset")
}
