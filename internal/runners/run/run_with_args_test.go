package run

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setupProjectWithScriptsExpectingArgs(t *testing.T, cmdName string) *projectfile.Project {
	if runtime.GOOS == "windows" {
		// Windows supports bash, but for the purpose of this test we only want to test cmd.exe, so ensure
		// that we run with cmd.exe even if the test is ran from bash
		os.Unsetenv("SHELL")
	} else {
		os.Setenv("SHELL", "bash")
	}

	tmpfile, err := ioutil.TempFile("", "testRunCommand")
	require.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: %s
    standalone: true
    value: |
      echo "ARGS|${1}|${2}|${3}|${4}|"`, cmdName)
	} else {
		contents = fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: %s
    standalone: true
    value: |
      echo "ARGS|%%1|%%2|%%3|%%4|"`, cmdName)
	}
	err = yaml.Unmarshal([]byte(contents), project)

	require.Nil(t, err, "error unmarshalling project yaml")
	return project
}

func captureExecCommand(t *testing.T, tmplCmdName, cmdName string, cmdArgs []string) (string, error) {
	failures.ResetHandled()

	project := setupProjectWithScriptsExpectingArgs(t, tmplCmdName)
	project.Persist()
	defer projectfile.Reset()

	var err error
	outStr, outErr := osutil.CaptureStdout(func() {
		err = run(outputhelper.NewCatcher(), cmdName, cmdArgs)
	})
	require.NoError(t, outErr, "error capturing stdout")
	require.NoError(t, failures.Handled(), "No failures handled")

	return outStr, err
}

func assertExecCommandProcessesArgs(t *testing.T, tmplCmdName, cmdName string, cmdArgs []string, expectedStdout string) {
	outStr, err := captureExecCommand(t, tmplCmdName, cmdName, cmdArgs)

	require.NoError(t, err, "unexpected error occurred")

	assert.Contains(t, outStr, expectedStdout)
}

func assertExecCommandFails(t *testing.T, tmplCmdName, cmdName string, cmdArgs []string, failureType *failures.FailureType) {
	_, err := captureExecCommand(t, tmplCmdName, cmdName, cmdArgs)
	require.Error(t, err, "run with error")

	fail, ok := err.(*failures.Failure)
	require.True(t, ok, "error must be failure (for now)")
	assert.Equal(t, failureType, fail.Type, "run error: No failure occurred")
}

func TestArgs_NoArgsProvided(t *testing.T) {
	assertExecCommandFails(t, "junk", "", []string{}, failures.FailUserInput)
}

func TestArgs_NoCmd_OnlyDash(t *testing.T) {
	assertExecCommandFails(t, "junk", "--", []string{}, FailScriptNotDefined)
}

func TestArgs_NameAndDashOnly(t *testing.T) {
	assertExecCommandProcessesArgs(t, "foo", "foo", []string{"--"}, "ARGS|--||||")
}

func TestArgs_MultipleArgs_NoDash(t *testing.T) {
	assertExecCommandProcessesArgs(t, "bar", "bar", []string{"baz", "bee"}, "ARGS|baz|bee|||")
}

func TestArgs_NoCmd_DashAsScriptName(t *testing.T) {
	assertExecCommandFails(t, "junk", "--", []string{"foo", "geez"}, FailScriptNotDefined)
}

func TestArgs_NoCmd_FlagAsScriptName(t *testing.T) {
	assertExecCommandFails(t, "junk", "-f", []string{"--foo", "geez"}, FailScriptNotDefined)
}

func TestArgs_WithCmd_AllArgsAfterDash(t *testing.T) {
	assertExecCommandProcessesArgs(t, "release", "release", []string{"--", "the", "kraken"}, "ARGS|--|the|kraken||")
}

func TestArgs_WithCmd_WithArgs_NoDash(t *testing.T) {
	assertExecCommandProcessesArgs(t, "release", "release", []string{"the", "kraken"}, "ARGS|the|kraken|||")
}

func TestArgs_WithCmd_WithArgs_BeforeAndAfterDash(t *testing.T) {
	assertExecCommandProcessesArgs(t, "foo", "foo", []string{"bar", "--", "bees", "wax"}, "ARGS|bar|--|bees|wax|")
}

func TestArgs_WithCmd_WithFlags_BeforeAndAfterDash(t *testing.T) {
	assertExecCommandProcessesArgs(t, "foo", "foo", []string{"--bar", "--", "bees", "--wax"}, "ARGS|--bar|--|bees|--wax|")
}
