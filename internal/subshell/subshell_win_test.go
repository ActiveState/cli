// +build windows

package subshell

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBash(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", `C:\Program Files\bash.exe`)
	subs, err := Get()
	require.NoError(t, err)
	assert.Equal(t, `C:\Program Files\bash.exe`, subs.Binary())

}

func TestBashDontEscapeSpace(t *testing.T) {
	setup(t)

	// Reproduce bug in which paths are being incorrectly escaped on windows
	os.Setenv("SHELL", `C:\Program\ Files\bash.exe`)
	subs, err := Get()
	require.NoError(t, err)
	assert.Equal(t, `C:\Program Files\bash.exe`, subs.Binary())
}

func TestRunCommandNoProjectEnv(t *testing.T) {
	pfile := &projectfile.Project{}
	pfile.Persist()
	os.Setenv("ComSpec", "C:\\WINDOWS\\system32\\cmd.exe")
	os.Setenv("ACTIVESTATE_PROJECT", "SHOULD NOT BE SET")
	os.Unsetenv("SHELL")

	subs, err := Get()
	assert.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "testRunCommand")
	assert.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	out, err := osutil.CaptureStdout(func() {
		_, err := subs.Run(`echo --EMPTY-- %ACTIVESTATE_PROJECT% --EMPTY--`)
		require.NoError(t, err)
	})
	require.NoError(t, err)
	assert.Contains(t, out, "--EMPTY--  --EMPTY--",
		strings.TrimSpace(out),
		"Should not echo anything cause the ACTIVESTATE_PROJECT should be undefined by the run command")

	projectfile.Reset()
}

func TestRunCommandError(t *testing.T) {
	pfile := &projectfile.Project{}
	pfile.Persist()

	os.Unsetenv("SHELL")

	subs, err := Get()
	assert.NoError(t, err)

	code, err := subs.Run("some-command-that-doesnt-exist")
	assert.Equal(t, 1, code, "Returns exit code 1")
	assert.Error(t, err, "Returns an error")

	code, err = subs.Run("exit 1")
	assert.Equal(t, 1, code, "Returns exit code 1")
	assert.Error(t, err, "Returns an error")

	projectfile.Reset()
}
