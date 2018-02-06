package virtualenvironment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
}

func TestActivate(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", "bash")
	venv, err := Activate()

	assert.NoError(t, err, "Should activate")

	assert.Equal(t, "bash", venv.Shell(), "Should detect bash as the shell")
	assert.True(t, venv.IsActive(), "Subshell should be active")

	err = venv.Deactivate()
	assert.NoError(t, err, "Should deactivate")

	assert.False(t, venv.IsActive(), "Subshell should be inactive")
}

func TestActivateFailures(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", "foo")
	_, err := Activate()

	assert.Error(t, err, "Should produce an error because of unsupported shell")
}
