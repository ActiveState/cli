//go:build !windows
// +build !windows

package subshell

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
)

func TestRunCommandNoProjectEnv(t *testing.T) {
	os.Setenv("SHELL", "bash")
	os.Setenv("ACTIVESTATE_PROJECT", "SHOULD NOT BE SET")

	cfg, err := config.New()
	require.NoError(t, err)
	subs := New(cfg)

	data := []byte("#!/usr/bin/env bash\necho $ACTIVESTATE_PROJECT")
	filename, err := fileutils.WriteTempFileToDir("", "testRunCommand", data, 0700)
	require.NoError(t, err)
	defer os.Remove(filename)

	out, err := osutil.CaptureStdout(func() {
		rerr := subs.Run(filename)
		require.NoError(t, rerr)
	})
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out), "Should not echo anything cause the ACTIVESTATE_PROJECT should be undefined by the run command")
}

func TestRunCommandError(t *testing.T) {
	os.Setenv("SHELL", "bash")

	cfg, err := config.New()
	require.NoError(t, err)
	subs := New(cfg)

	err = subs.Run("some-file-that-doesnt-exist")
	assert.Error(t, err, "Returns an error")

	data := []byte("#!/usr/bin/env bash\nexit 2")
	filename, err := fileutils.WriteTempFileToDir("", "testRunCommand", data, 0700)
	require.NoError(t, err)
	defer os.Remove(filename)

	err = subs.Run(filename)
	require.Error(t, err, "Returns an error")
	var eerr interface{ ExitCode() int }
	require.True(t, errors.As(err, &eerr), "Error is exec exit error")
	assert.Equal(t, eerr.ExitCode(), 2, "Returns exit code 2")
}
