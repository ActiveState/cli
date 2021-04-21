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

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
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

	cfg, err := config.Get()
	require.NoError(t, err)
	subs := New(cfg)

	filename, err := fileutils.WriteTempFile("", "testRunCommand*.bat", data, 0700)
	require.NoError(t, err)
	defer os.Remove(filename)

	out, err := osutil.CaptureStdout(func() {
		rerr := subs.Run(filename)
		require.NoError(t, rerr)
	})
	require.NoError(t, err)

	trimmed := strings.TrimSpace(out)
	assert.Equal(t, "Hello", trimmed[len(trimmed)-len("Hello"):])

	projectfile.Reset()
}
