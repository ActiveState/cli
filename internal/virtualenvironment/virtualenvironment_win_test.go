// +build windows

package virtualenvironment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnvSlice_NoPath(t *testing.T) {
	setup(t)
	defer teardown()

	venv := Init()
	env, err := venv.GetEnvSlice(true)
	require.NoError(t, err)
	assert.NotContains(t, env, "Path")
}
