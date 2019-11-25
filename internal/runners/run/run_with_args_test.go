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
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
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

func captureExecCommand(t *testing.T, cmdName string, cmdArgs []string) (int, string) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := setupProjectWithScriptsExpectingArgs(t, cmdName)
	project.Persist()
	defer projectfile.Reset()

	Cc := Command.GetCobraCmd()
	// without this Unregister call, positional arg state is persisted between tests
	defer Command.Unregister()

	Cc.SetArgs(cmdArgs)

	var outStr string
	Command.Exiter = exiter.Exit
	exitCode := exiter.WaitForExit(func() {
		var outErr error
		outStr, outErr = osutil.CaptureStdout(func() {
			err := Command.Execute()
			require.NoError(t, err, "error executing command")
		})
		require.NoError(t, outErr, "error capturing stdout")
	})
	return exitCode, outStr
}

func assertExecCommandProcessesArgs(t *testing.T, cmdName string, cmdArgs []string, expectedStdout string) {
	exitCode, outStr := captureExecCommand(t, cmdName, cmdArgs)

	require.Equal(t, -1, exitCode, "Exits normally")
	require.Nil(t, failures.Handled(), "unexpected failure occurred")
	assert.Contains(t, outStr, expectedStdout)
}

func assertExecCommandFails(t *testing.T, cmdName string, cmdArgs []string, failureType *failures.FailureType) {
	exitCode, _ := captureExecCommand(t, cmdName, cmdArgs)

	require.Equal(t, 1, exitCode, "Exits with code 1")
	handled := failures.Handled()
	require.NotNil(t, handled, "expected a failure")
	assert.Equal(t, failureType, handled.(*failures.Failure).Type, "No failure occurred")
}

func TestArgs_NoArgsProvided(t *testing.T) {
	// state run
	assertExecCommandFails(t, "run", []string{}, failures.FailUserInput)
}

func TestArgs_NoCmd_OnlyDash(t *testing.T) {
	// state run --
	assertExecCommandFails(t, "run", []string{"--"}, failures.FailUserInput)
}

func TestArgs_NameAndDashOnly(t *testing.T) {
	// state run foo --
	assertExecCommandProcessesArgs(t, "foo", []string{"foo", "--"}, "ARGS|--||||")
}

func TestArgs_MultipleArgs_NoDash(t *testing.T) {
	// state run bar baz bee
	assertExecCommandProcessesArgs(t, "bar", []string{"bar", "baz", "bee"}, "ARGS|baz|bee|||")
}

func TestArgs_NoCmd_AllArgsAfterDash(t *testing.T) {
	// state run -- foo geez
	assertExecCommandFails(t, "run", []string{"--", "foo", "geez"}, failures.FailUserInput)
}

func TestArgs_NoCmd_FlagAsFirstArg(t *testing.T) {
	// state run -- foo geez
	assertExecCommandFails(t, "run", []string{"-f", "--foo", "geez"}, failures.FailUserInput)
}

func TestArgs_WithCmd_AllArgsAfterDash(t *testing.T) {
	// state run release -- the kraken
	assertExecCommandProcessesArgs(t, "release", []string{"release", "--", "the", "kraken"}, "ARGS|--|the|kraken||")
}

func TestArgs_WithCmd_WithArgs_NoDash(t *testing.T) {
	// state run release the kraken
	assertExecCommandProcessesArgs(t, "release", []string{"release", "the", "kraken"}, "ARGS|the|kraken|||")
}

func TestArgs_WithCmd_WithArgs_BeforeAndAfterDash(t *testing.T) {
	// state run foo bar -- bees wax
	assertExecCommandProcessesArgs(t, "foo", []string{"foo", "bar", "--", "bees", "wax"}, "ARGS|bar|--|bees|wax|")
}

func TestArgs_WithCmd_WithFlags_BeforeAndAfterDash(t *testing.T) {
	// state run foo --bar -- bees --wax
	assertExecCommandProcessesArgs(t, "foo", []string{"foo", "--bar", "--", "bees", "--wax"}, "ARGS|--bar|--|bees|--wax|")
}
