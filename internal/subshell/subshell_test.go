package subshell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
}

func TestActivate(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", "bash")
	os.Setenv("ComSpec", "cmd.exe")
	subs, fail := Activate()

	assert.NoError(t, fail.ToError(), "Should activate")

	assert.NotEqual(t, "", subs.Shell(), "Should detect a shell")
	assert.True(t, subs.IsActive(), "Subshell should be active")

	fail = subs.Deactivate()
	assert.NoError(t, fail.ToError(), "Should deactivate")

	assert.False(t, subs.IsActive(), "Subshell should be inactive")
}

func TestActivateCmdExists(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", "bash")
	os.Setenv("ComSpec", "cmd.exe")

	filename := "debug"
	if runtime.GOOS == "windows" {
		filename = "debug.exe"
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL, 0700)
	assert.NoError(t, err, "Should be able to create executable file")
	defer os.Remove(f.Name())

	err = f.Close()
	assert.NoError(t, err, "Could no close file")

	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	wd, err := os.Getwd()
	assert.NoError(t, err, "Could not get current working directory")

	err = os.Setenv("PATH", wd)
	assert.NoError(t, err, "Could not set PATH")

	out, err := osutil.CaptureStdout(func() {
		subs, fail := Activate()
		assert.NoError(t, fail.ToError(), "Should activate")

		assert.NotEqual(t, "", subs.Shell(), "Should detect a shell")
		assert.True(t, subs.IsActive(), "Subshell should be active")

		fail = subs.Deactivate()
		assert.NoError(t, fail.ToError(), "Should deactivate")

		assert.False(t, subs.IsActive(), "Subshell should be inactive")
	})
	require.NoError(t, err)

	assert.Equal(t, locale.Tr("warn_script_name_in_use", "debug", "debug", "project", "project_debug"), strings.TrimSuffix(out, "\n"), "output should match")
}

func TestActivateFailures(t *testing.T) {
	setup(t)

	shell := os.Getenv("SHELL")
	comspec := os.Getenv("ComSpec")

	os.Setenv("SHELL", "foo")
	os.Setenv("ComSpec", "foo")
	_, err := Activate()
	os.Setenv("SHELL", shell)
	os.Setenv("ComSpec", comspec)

	assert.Error(t, err, "Should produce an error because of unsupported shell")
}

func TestRunCommand(t *testing.T) {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	data := []byte("echo Hello")
	if runtime.GOOS == "windows" {
		// Windows supports bash, but for the purpose of this test we only want to test cmd.exe, so ensure
		// that we run with cmd.exe even if the test is ran from bash
		os.Unsetenv("SHELL")
	} else {
		data = append([]byte("#!/usr/bin/env bash\n"), data...)
		os.Setenv("SHELL", "bash")
	}

	subs, fail := Get()
	require.NoError(t, fail.ToError())

	filename, fail := fileutils.WriteTempFile("", "testRunCommand*.bat", data, 0700)
	require.NoError(t, fail.ToError())
	defer os.Remove(filename)

	out, err := osutil.CaptureStdout(func() {
		_, err := subs.Run(filename)
		require.NoError(t, err)
	})
	require.NoError(t, err)

	trimmed := strings.TrimSpace(out)
	assert.Equal(t, "Hello", trimmed[len(trimmed)-len("Hello"):])

	projectfile.Reset()
}

func TestIsActivated(t *testing.T) {
	assert.False(t, IsActivated())
}
