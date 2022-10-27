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
	// Fetch current home directory.
	osHomeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	// Verify user.HomeDir() returns the current home directory.
	userHomeDir, err := HomeDir()
	require.NoError(t, err)
	assert.Equal(t, userHomeDir, osHomeDir)

	// Verify ACTIVESTATE_HOME overrides current home directory.
	os.Setenv(constants.HomeEnvVarName, "override")
	userHomeDir, err = HomeDir()
	require.NoError(t, err)
	assert.Equal(t, userHomeDir, "override")
	os.Unsetenv(constants.HomeEnvVarName)

	// Verify lack of HOME and ACTIVESTATE_HOME shows nice error message.
	if runtime.GOOS != "windows" {
		os.Unsetenv("HOME")
		defer func() { os.Setenv("HOME", osHomeDir) }()
	} else {
		os.Unsetenv("USERPROFILE")
		defer func() { os.Setenv("USERPROFILE", osHomeDir) }()
	}
	userHomeDir, err = HomeDir()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HOME environment variable is unset")
}
