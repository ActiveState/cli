// +build !windows

package subshell

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func TestActivateZsh(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", "zsh")
	venv, fail := Activate()

	assert.NoError(t, fail.ToError(), "Should activate")

	assert.NotEqual(t, "", venv.Shell(), "Should detect a shell")
	assert.True(t, venv.IsActive(), "Subshell should be active")

	fail = venv.Deactivate()
	assert.NoError(t, fail.ToError(), "Should deactivate")

	assert.False(t, venv.IsActive(), "Subshell should be inactive")
}

func TestRunCommandNoProjectEnv(t *testing.T) {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	os.Setenv("SHELL", "bash")
	os.Setenv("ACTIVESTATE_PROJECT", "SHOULD NOT BE SET")

	subs, fail := Get()
	require.NoError(t, fail.ToError())

	data := []byte("#!/usr/bin/env bash\necho $ACTIVESTATE_PROJECT")
	filename, fail := fileutils.WriteTempFile("", "testRunCommand", data, 0700)
	require.NoError(t, fail.ToError())
	defer os.Remove(filename)

	out, err := osutil.CaptureStdout(func() {
		_, err := subs.Run(filename)
		require.NoError(t, err)
	})
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out), "Should not echo anything cause the ACTIVESTATE_PROJECT should be undefined by the run command")

	projectfile.Reset()
}

func TestRunCommandError(t *testing.T) {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	os.Setenv("SHELL", "bash")

	subs, fail := Get()
	require.NoError(t, fail.ToError())

	code, err := subs.Run("some-file-that-doesnt-exist")
	assert.Error(t, err, "Returns an error")
	assert.Equal(t, 1, code, "Returns exit code 1")

	data := []byte("#!/usr/bin/env bash\nexit 2")
	filename, fail := fileutils.WriteTempFile("", "testRunCommand", data, 0700)
	require.NoError(t, fail.ToError())
	defer os.Remove(filename)

	code, err = subs.Run(filename)
	assert.Error(t, err)
	assert.Equal(t, 2, code, "Returns exit code 2")

	projectfile.Reset()
}
