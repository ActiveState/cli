package subshell

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivateZsh(t *testing.T) {
	setup(t)
	var wg sync.WaitGroup

	os.Setenv("SHELL", "zsh")
	venv, err := Activate(&wg)

	assert.NoError(t, err, "Should activate")

	assert.NotEqual(t, "", venv.Shell(), "Should detect a shell")
	assert.True(t, venv.IsActive(), "Subshell should be active")

	err = venv.Deactivate()
	assert.NoError(t, err, "Should deactivate")

	assert.False(t, venv.IsActive(), "Subshell should be inactive")
}
