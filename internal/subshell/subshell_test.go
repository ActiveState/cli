package subshell

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
)

func TestRunCommand(t *testing.T) {
	data := []byte("echo Hello")
	if runtime.GOOS == "windows" {
		// Windows supports bash, but for the purpose of this test we only want to test cmd.exe, so ensure
		// that we run with cmd.exe even if the test is ran from bash
		os.Unsetenv("SHELL")
	} else {
		data = append([]byte("#!/usr/bin/env bash\n"), data...)
		os.Setenv("SHELL", "bash")
	}

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	subs := New(cfg)

	filename, err := fileutils.WriteTempFileToDir("", "testRunCommand*.bat", data, 0700)
	require.NoError(t, err)
	defer os.Remove(filename)

	out, err := osutil.CaptureStdout(func() {
		rerr := subs.Run(filename)
		require.NoError(t, rerr)
	})
	require.NoError(t, err)

	trimmed := strings.TrimSpace(out)
	assert.Equal(t, "Hello", trimmed[len(trimmed)-len("Hello"):])
}
