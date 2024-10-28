//go:build windows
// +build windows

package subshell

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "test"))
	assert.NoError(t, err, "Should change to test directory")
}

func TestBash(t *testing.T) {
	setup(t)

	shellPath := `C:\Program Files\Git\usr\bin\bash.exe`
	os.Setenv("SHELL", shellPath)
	cfg, err := config.New()
	require.NoError(t, err)
	subs := New(cfg)
	assert.Equal(t, shellPath, subs.Binary())

}

func TestBashDontEscapeSpace(t *testing.T) {
	setup(t)

	// Reproduce bug in which paths are being incorrectly escaped on windows
	os.Setenv("SHELL", `C:\Program\ Files\Git\usr\bin\bash.exe`)
	cfg, err := config.New()
	require.NoError(t, err)
	subs := New(cfg)
	assert.Equal(t, `C:\Program Files\Git\usr\bin\bash.exe`, subs.Binary())
}

func TestRunCommandError(t *testing.T) {
	os.Unsetenv("SHELL")

	cfg, err := config.New()
	require.NoError(t, err)
	subs := New(cfg)

	err = subs.Run("some-file-that-doesnt-exist.bat")
	assert.Error(t, err, "Returns an error")

	data := []byte("exit 2")
	filename, err := fileutils.WriteTempFileToDir("", "test*.bat", data, 0700)
	require.NoError(t, err)
	defer os.Remove(filename)

	err = subs.Run(filename)
	var eerr interface{ ExitCode() int }
	require.True(t, errors.As(err, &eerr), "Error is exec exit error")
	assert.Equal(t, eerr.ExitCode(), 2, "Returns exit code 2")
}
