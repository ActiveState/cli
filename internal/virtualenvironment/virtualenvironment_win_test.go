// +build windows

package virtualenvironment

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnvSlice_NoPath(t *testing.T) {
	setup(t)
	defer teardown()

	os.Setenv(constants.DisableRuntime, "true")
	defer os.Unsetenv(constants.DisableRuntime)

	venv := Init()
	env, err := venv.GetEnvSlice(true)
	require.NoError(t, err)
	assert.NotContains(t, env, "Path")
}
