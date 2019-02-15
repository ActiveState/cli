package run

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/osutil"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func setupProjectWithScriptsExpectingArgs(t *testing.T, cmdName string) *projectfile.Project {
	os.Setenv("SHELL", "bash")

	tmpfile, err := ioutil.TempFile("", "testRunCommand")
	require.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = fmt.Sprintf(`
scripts:
  - name: %s
    standalone: true
    value: |
      echo "ARGS|${1}|${2}|${3}|${4}|"`, cmdName)
	} else {
		contents = fmt.Sprintf(`
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

func captureExecCommand(t *testing.T, cmdName string, cmdArgs []string) string {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := setupProjectWithScriptsExpectingArgs(t, cmdName)
	project.Persist()
	defer projectfile.Reset()

	Cc := Command.GetCobraCmd()
	// without this Unregister call, positional arg state is persisted between tests
	defer Command.Unregister()

	Cc.SetArgs(cmdArgs)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = Command.Execute()
	})
	require.NoError(t, execErr, "error executing command")
	require.NoError(t, outErr, "error capturing stdout")
	return outStr
}

func assertExecCommandProcessesArgs(t *testing.T, cmdName string, cmdArgs []string, expectedStdout string) {
	outStr := captureExecCommand(t, cmdName, cmdArgs)

	require.Nil(t, failures.Handled(), "unexpected failure occurred")
	assert.Contains(t, outStr, expectedStdout)
}

func assertExecCommandFails(t *testing.T, cmdName string, cmdArgs []string, failureType *failures.FailureType) {
	captureExecCommand(t, cmdName, cmdArgs)

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
