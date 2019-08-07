// +build windows

package subshell

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func TestBash(t *testing.T) {
	setup(t)

	os.Setenv("SHELL", `C:\Program Files\bash.exe`)
	subs, fail := Get()
	require.NoError(t, fail.ToError())
	assert.Equal(t, `C:\Program Files\bash.exe`, subs.Binary())

}

func TestBashDontEscapeSpace(t *testing.T) {
	setup(t)

	// Reproduce bug in which paths are being incorrectly escaped on windows
	os.Setenv("SHELL", `C:\Program\ Files\bash.exe`)
	subs, fail := Get()
	require.NoError(t, fail.ToError())
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

	subs, fail := Get()
	assert.NoError(t, fail.ToError())

	tmpfile, err := ioutil.TempFile("", "testRunCommand*.bat")
	assert.NoError(t, err)
	tmpfile.WriteString("echo --EMPTY-- %ACTIVESTATE_PROJECT% --EMPTY--")
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0755)
	defer os.Remove(tmpfile.Name())

	out, err := osutil.CaptureStdout(func() {
		_, err := subs.Run(tmpfile.Name())
		require.NoError(t, err)
	})
	require.NoError(t, err)
	assert.Contains(t, out, "--EMPTY--  --EMPTY--",
		strings.TrimSpace(out),
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

	subs, fail := Get()
	assert.NoError(t, fail.ToError())

	code, err := subs.Run("some-file-that-doesnt-exist")
	assert.Equal(t, 1, code, "Returns exit code 1")
	assert.Error(t, err, "Returns an error")

	tmpfile, err := ioutil.TempFile("", "testRunCommand*.bat")
	assert.NoError(t, err)
	tmpfile.WriteString("exit 1")
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0755)
	defer os.Remove(tmpfile.Name())

	code, err = subs.Run(tmpfile.Name())
	assert.Equal(t, 1, code, "Returns exit code 1")
	assert.Error(t, err, "Returns an error")

	projectfile.Reset()
}
