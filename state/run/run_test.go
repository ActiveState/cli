package run

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	rtMock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func init() {
	mock := rtMock.Init()
	mock.MockFullRuntime()
}

func TestRunStandaloneCommand(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: echo foo
    standalone: true
  `)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: cmd /C echo foo
    standalone: true
  `)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestEnvIsSet(t *testing.T) {
	if runtime.GOOS == "windows" {
		// For some reason this test hangs on Windows when ran via CI. I cannot reproduce the issue when manually invoking the
		// test. Seeing as there isnt really any Windows specific logic being tested here I'm just disabling the test on Windows
		// as it's not worth the time and effort to debug.
		return
	}
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: printenv
  `)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: cmd.exe /C SET
  `)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	os.Setenv("TEST_KEY_EXISTS", "true")
	os.Setenv(constants.DisableRuntime, "true")

	ex := exiter.New()
	var exitCode int
	Command.Exiter = ex.Exit
	out := capturer.CaptureOutput(func() {
		exitCode = ex.WaitForExit(func() {
			err = Command.Execute()
		})
	})

	assert.Equal(t, -1, exitCode, fmt.Sprintf("Exited with code %d, output: %s", exitCode, out))
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	assert.Contains(t, out, constants.ActivatedStateEnvVarName)
	assert.Contains(t, out, "TEST_KEY_EXISTS")
}

func TestRunNoProjectInheritance(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: echo $ACTIVESTATE_PROJECT
    standalone: true
`)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: echo %ACTIVESTATE_PROJECT%
    standalone: true
`)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})

	out, err := osutil.CaptureStdout(func() {
		err := Command.Execute()
		require.NoError(t, err)
	})
	require.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	assert.Contains(t, out, "Running user-defined script: run")
}

func TestRunMissingCommandName(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{""})

	ex := exiter.New()
	Command.Exiter = ex.Exit
	exitCode := ex.WaitForExit(func() {
		Command.Execute()
	})
	assert.Equal(t, 1, exitCode, "Exited with code 1")

	handled := failures.Handled()
	require.NotNil(t, handled, "expected a failure")
	assert.Equal(t, failures.FailUserInput, handled.(*failures.Failure).Type, "Use input failure occurred")
}

func TestRunUnknownCommandName(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"unknown"})
	ex := exiter.New()
	Command.Exiter = ex.Exit
	exitCode := ex.WaitForExit(func() {
		Command.Execute()
	})
	assert.Equal(t, 1, exitCode, "Exited with code 1")
}

func TestRunUnknownCommand(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    value: whatever
    standalone: true
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Command.Register()
	Command.Exiter = exiter.Exit

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	exitCode := exiter.WaitForExit(func() { Command.Execute() })

	assert.NotEqual(t, 0, exitCode, "Execution caused exit")
	assert.Error(t, failures.Handled(), "Failure occurred")
}

func TestRunActivatedCommand(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	// Prepare an empty activated environment.
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	datadir := config.ConfigPath()
	os.RemoveAll(filepath.Join(datadir, "virtual"))
	os.RemoveAll(filepath.Join(datadir, "packages"))
	os.RemoveAll(filepath.Join(datadir, "languages"))
	os.RemoveAll(filepath.Join(datadir, "artifacts"))

	// Setup the project.
	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    standalone: true
    value: echo foo`)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    standalone: true
    value: cmd /C echo foo`)
	}
	err = yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	// Run the command.
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	failures.ResetHandled()
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	// Reset.
	projectfile.Reset()
}

func TestPathProvidesExec(t *testing.T) {
	tf, err := ioutil.TempFile("", "t*.t")
	require.NoError(t, err)
	defer os.Remove(tf.Name())

	require.NoError(t, os.Chmod(tf.Name(), 0770))

	exec := path.Base(tf.Name())
	temp := path.Dir(tf.Name())

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	paths := []string{temp, home}
	pathStr := strings.Join(paths, string(os.PathListSeparator))

	assert.True(t, pathProvidesExec(temp, exec, path.Dir(tf.Name())))
	assert.True(t, pathProvidesExec(temp, exec, pathStr))
	assert.False(t, pathProvidesExec(temp, "junk", pathStr))
	assert.False(t, pathProvidesExec(temp, exec, ""))
}

func TestRun_Help(t *testing.T) {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--help"})
	outStr, err := osutil.CaptureStdout(func() {
		err := Cc.Execute()
		require.NoError(t, err, "error executing command")
	})
	require.NoError(t, err, "error capturing stdout")
	assert.Equal(t, Cc.UsageString(), strings.TrimSuffix(outStr, "\n"))
}
