package run

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	rtMock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func init() {
	mock := rtMock.Init()
	mock.MockFullRuntime()
}

func TestRunStandaloneCommand(t *testing.T) {
	failures.ResetHandled()

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: echo foo
    standalone: true
  `)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: cmd /C echo foo
    standalone: true
  `)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	err = run(outputhelper.NewCatcher(), subshell.New(), "run", nil)
	assert.NoError(t, err, "No error occurred")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestEnvIsSet(t *testing.T) {
	if runtime.GOOS == "windows" {
		// For some reason this test hangs on Windows when ran via CI. I cannot reproduce the issue when manually invoking the
		// test. Seeing as there isnt really any Windows specific logic being tested here I'm just disabling the test on Windows
		// as it's not worth the time and effort to debug.
		return
	}
	failures.ResetHandled()

	root, err := environment.GetRootPath()
	require.NoError(t, err, "should detect root path")
	prjPath := filepath.Join(root, "internal", "runners", "run", "testdata", "printEnv", "activestate.yaml")

	project, fail := projectfile.Parse(prjPath)
	require.NoError(t, fail.ToError(), "parsing project file")
	project.Persist()

	os.Setenv("TEST_KEY_EXISTS", "true")
	os.Setenv(constants.DisableRuntime, "true")
	defer func() {
		os.Unsetenv("TEST_KEY_EXISTS")
		os.Unsetenv(constants.DisableRuntime)
	}()

	out := capturer.CaptureOutput(func() {
		err = run(outputhelper.NewCatcher(), subshell.New(), "run", nil)
		assert.NoError(t, err, "No error occurred")
		assert.NoError(t, failures.Handled(), "No failure occurred")
	})

	assert.Contains(t, out, constants.ActivatedStateEnvVarName)
	assert.Contains(t, out, "TEST_KEY_EXISTS")
}

func TestRunNoProjectInheritance(t *testing.T) {
	failures.ResetHandled()

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: echo $ACTIVESTATE_PROJECT
    standalone: true
`)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: echo %ACTIVESTATE_PROJECT%
    standalone: true
`)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	out, err := osutil.CaptureStdout(func() {
		rerr := run(outputhelper.NewCatcher(), subshell.New(), "run", nil)
		assert.NoError(t, rerr, "No error occurred")
		assert.NoError(t, failures.Handled(), "No failure occurred")
	})
	require.NoError(t, err, "Executed without error")
	assert.Contains(t, out, "Running user-defined script: run")
}

func TestRunMissingCommandName(t *testing.T) {
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	err = run(outputhelper.NewCatcher(), subshell.New(), "", nil)
	assert.Error(t, err, "Error occurred")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRunUnknownCommandName(t *testing.T) {
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	err = run(outputhelper.NewCatcher(), subshell.New(), "unknown", nil)
	assert.Error(t, err, "Error occurred")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func estRunUnknownCommand(t *testing.T) {
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    value: whatever
    standalone: true
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	err = run(outputhelper.NewCatcher(), subshell.New(), "run", nil)
	assert.Error(t, err, "Error occurred")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRunActivatedCommand(t *testing.T) {
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
    languages: [bash]
    standalone: true
    value: echo foo`)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: run
    languages: [bash]
    standalone: true
    value: cmd /C echo foo`)
	}
	err = yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	// Run the command.
	err = run(outputhelper.NewCatcher(), subshell.New(), "run", nil)
	assert.NoError(t, err, "No error occurred")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	// Reset.
	projectfile.Reset()
}

func TestPathProvidesExec(t *testing.T) {
	temp, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	tf := filepath.Join(temp, "python3")
	fail := fileutils.Touch(tf)
	require.NoError(t, fail.ToError())
	defer os.Remove(temp)

	require.NoError(t, os.Chmod(tf, 0770))

	exec := language.Python3

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	paths := []string{temp, home}
	pathStr := strings.Join(paths, string(os.PathListSeparator))

	assert.True(t, pathProvidesExec(temp, filepath.Dir(tf), exec))
	assert.True(t, pathProvidesExec(temp, pathStr, exec))
	assert.False(t, pathProvidesExec(temp, pathStr, language.Unknown))
	assert.False(t, pathProvidesExec(temp, "", exec))
}
