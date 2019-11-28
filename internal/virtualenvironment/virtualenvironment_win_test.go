package virtualenvironment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvSlice_NoPath(t *testing.T) {
	setup(t)
	defer teardown()

	venv := Init()
	env := venv.GetEnvSlice(true)
	assert.NotContains(t, env, "Path")
}
