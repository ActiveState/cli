// +build windows

package subshell

import (
	"errors"
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

func TestBash(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", `C:\Program Files\bash.exe`)
	subs := New()
	assert.Equal(t, `C:\Program Files\bash.exe`, subs.Binary())

}

func TestBashDontEscapeSpace(t *testing.T) {
	setup(t)

	// Reproduce bug in which paths are being incorrectly escaped on windows
	os.Setenv("SHELL", `C:\Program\ Files\bash.exe`)
	subs := New()
	assert.Equal(t, `C:\Program Files\bash.exe`, subs.Binary())
}

func TestRunCommandNoProjectEnv(t *testing.T) {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()
	os.Setenv("ComSpec", "C:\\WINDOWS\\system32\\cmd.exe")
	os.Setenv("ACTIVESTATE_PROJECT", "SHOULD NOT BE SET")
	os.Unsetenv("SHELL")

	subs := New()

	data := []byte("echo --EMPTY-- %ACTIVESTATE_PROJECT% --EMPTY--")
	filename, fail := fileutils.WriteTempFile("", "test*.bat", data, 0700)
	require.NoError(t, fail.ToError())
	defer os.Remove(filename)

	out, err := osutil.CaptureStdout(func() {
		rerr := subs.Run(filename)
		require.NoError(t, rerr)
	})
	require.NoError(t, err)
	assert.Contains(t, out, "--EMPTY--  --EMPTY--", strings.TrimSpace(out),
		"Should not echo anything cause the ACTIVESTATE_PROJECT should be undefined by the run command")

	projectfile.Reset()
}

func TestRunCommandError(t *testing.T) {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	os.Unsetenv("SHELL")

	subs := New()

	err := subs.Run("some-file-that-doesnt-exist.bat")
	assert.Error(t, err, "Returns an error")

	data := []byte("exit 2")
	filename, fail := fileutils.WriteTempFile("", "test*.bat", data, 0700)
	require.NoError(t, fail.ToError())
	defer os.Remove(filename)

	err = subs.Run(filename)
	var eerr interface{ ExitCode() int }
	require.True(t, errors.As(err, &eerr), "Error is exec exit error")
	assert.Equal(t, eerr.ExitCode(), 2, "Returns exit code 2")

	projectfile.Reset()
}
