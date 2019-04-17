// +build !windows

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

func TestRunCommandNoProjectEnv(t *testing.T) {
	pfile := &projectfile.Project{}
	pfile.Persist()

	os.Setenv("SHELL", "bash")
	os.Setenv("ACTIVESTATE_PROJECT", "SHOULD NOT BE SET")

	subs, err := Get()
	assert.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "testRunCommand")
	assert.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	out, err := osutil.CaptureStdout(func() {
		_, err := subs.Run(`echo $ACTIVESTATE_PROJECT`)
		require.NoError(t, err)
	})
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out), "Should not echo anything cause the ACTIVESTATE_PROJECT should be undefined by the run command")

	projectfile.Reset()
}

func TestRunCommandError(t *testing.T) {
	pfile := &projectfile.Project{}
	pfile.Persist()

	os.Setenv("SHELL", "bash")

	subs, err := Get()
	assert.NoError(t, err)

	code, err := subs.Run("some-command-that-doesnt-exist")
	assert.Equal(t, 127, code, "Returns exit code 127")
	assert.Error(t, err, "Returns an error")

	code, err = subs.Run("exit 1")
	assert.Equal(t, 1, code, "Returns exit code 1")
	assert.Error(t, err, "Returns an error")

	projectfile.Reset()
}