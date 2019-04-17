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

func TestRunCommandNoProjectEnv(t *testing.T) {
	pfile := &projectfile.Project{}
	pfile.Persist()
	os.Setenv("ComSpec", "C:\\WINDOWS\\system32\\cmd.exe")
	os.Setenv("ACTIVESTATE_PROJECT", "SHOULD NOT BE SET")

	subs, err := Get()
	assert.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "testRunCommand")
	assert.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	out, err := osutil.CaptureStdout(func() {
		_, err := subs.Run(`echo %ACTIVESTATE_PROJECT%`)
		require.NoError(t, err)
	})
	require.NoError(t, err)
	assert.Equal(t, "C:\\Users\\cgcho\\Projects\\ActiveState\\cli\\test>echo  \r\nECHO is on.",
		strings.TrimSpace(out),
		"Should not echo anything cause the ACTIVESTATE_PROJECT should be undefined by the run command")

	projectfile.Reset()
}

func TestRunCommandError(t *testing.T) {
	pfile := &projectfile.Project{}
	pfile.Persist()

	os.Setenv("SHELL", "bash")

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
