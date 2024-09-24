package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type ScriptRunSuite struct {
	tagsuite.Suite
}

func TestScriptRunSuite(t *testing.T) {
	suite.Run(t, new(ScriptRunSuite))
}

func (suite *ScriptRunSuite) TestRunStandaloneCommand() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	pjfile := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    value: echo foo
    standalone: true
  `)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    value: cmd.exe /C echo foo
    standalone: true
  `)
	}
	err = yaml.Unmarshal([]byte(contents), pjfile)
	assert.Nil(t, err, "Unmarshalled YAML")

	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	scriptRun := scriptrun.New(primer.New(proj, auth, outputhelper.NewCatcher(), subshell.New(cfg), cfg, blackhole.New()))
	script, err := proj.ScriptByName("run")
	require.NoError(t, err)
	err = scriptRun.Run(script, []string{})
	assert.NoError(t, err, "No error occurred")
}

func (suite *ScriptRunSuite) TestEnvIsSet() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	if runtime.GOOS == "windows" {
		// For some reason this test hangs on Windows when ran via CI. I cannot reproduce the issue when manually invoking the
		// test. Seeing as there isnt really any Windows specific logic being tested here I'm just disabling the test on Windows
		// as it's not worth the time and effort to debug.
		return
	}

	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	root, err := environment.GetRootPath()
	require.NoError(t, err, "should detect root path")
	prjPath := filepath.Join(root, "internal", "scriptrun", "test", "integration", "testdata", "printEnv", "activestate.yaml")

	pjfile, err := projectfile.Parse(prjPath)
	require.NoError(t, err, "parsing pjfile file")

	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)

	os.Setenv("TEST_KEY_EXISTS", "true")
	defer func() {
		os.Unsetenv("TEST_KEY_EXISTS")
	}()

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	cfg.Set(constants.AsyncRuntimeConfig, true)

	out := capturer.CaptureOutput(func() {
		scriptRun := scriptrun.New(primer.New(auth, outputhelper.NewCatcher(), subshell.New(cfg), proj, cfg, blackhole.New(), model.NewSvcModel("")))
		script, err := proj.ScriptByName("run")
		require.NoError(t, err, "Error: "+errs.JoinMessage(err))
		err = scriptRun.Run(script, nil)
		assert.NoError(t, err, "Error: "+errs.JoinMessage(err))
	})

	assert.Contains(t, out, constants.ActivatedStateEnvVarName)
	assert.Contains(t, out, "TEST_KEY_EXISTS")
}

func (suite *ScriptRunSuite) TestRunNoProjectInheritance() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	pjfile := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    value: echo $ACTIVESTATE_ACTIVATED
    standalone: true
`)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    value: echo %ACTIVESTATE_ACTIVATED%
    standalone: true
`)
	}
	err = yaml.Unmarshal([]byte(contents), pjfile)
	assert.Nil(t, err, "Unmarshalled YAML")

	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	out := outputhelper.NewCatcher()
	scriptRun := scriptrun.New(primer.New(auth, out, subshell.New(cfg), proj, cfg, blackhole.New()))
	script, err := proj.ScriptByName("run")
	fmt.Println(script)
	require.NoError(t, err)
	err = scriptRun.Run(script, nil)
	assert.NoError(t, err, "No error occurred")
}

func (suite *ScriptRunSuite) TestRunMissingScript() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	pjfile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    value: whatever
  `)
	err = yaml.Unmarshal([]byte(contents), pjfile)
	assert.Nil(t, err, "Unmarshalled YAML")

	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	scriptRun := scriptrun.New(primer.New(auth, outputhelper.NewCatcher(), subshell.New(cfg), proj, cfg, blackhole.New()))
	err = scriptRun.Run(nil, nil)
	assert.Error(t, err, "No error occurred")
}

func (suite *ScriptRunSuite) TestRunUnknownCommand() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	pjfile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    value: whatever
    standalone: true
  `)
	err = yaml.Unmarshal([]byte(contents), pjfile)
	assert.Nil(t, err, "Unmarshalled YAML")

	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	scriptRun := scriptrun.New(primer.New(auth, outputhelper.NewCatcher(), subshell.New(cfg), proj, cfg, blackhole.New()))
	script, err := proj.ScriptByName("run")
	require.NoError(t, err)
	err = scriptRun.Run(script, nil)
	assert.Error(t, err, "No error occurred")
}

func (suite *ScriptRunSuite) TestRunActivatedCommand() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	// Prepare an empty activated environment.
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "test"))
	assert.NoError(t, err, "Should change directory")

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	datadir := cfg.ConfigPath()
	os.RemoveAll(filepath.Join(datadir, "virtual"))
	os.RemoveAll(filepath.Join(datadir, "packages"))
	os.RemoveAll(filepath.Join(datadir, "languages"))
	os.RemoveAll(filepath.Join(datadir, "artifacts"))

	// Setup the pjfile.
	pjfile := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    standalone: true
    value: echo foo`)
	} else {
		contents = strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/pjfile"
scripts:
  - name: run
    standalone: true
    value: cmd /C echo foo`)
	}
	err = yaml.Unmarshal([]byte(contents), pjfile)
	assert.Nil(t, err, "Unmarshalled YAML")

	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)

	// Run the command.
	scriptRun := scriptrun.New(primer.New(auth, outputhelper.NewCatcher(), subshell.New(cfg), proj, cfg, blackhole.New()))
	script, err := proj.ScriptByName("run")
	require.NoError(t, err)
	err = scriptRun.Run(script, nil)
	assert.NoError(t, err, "No error occurred")
}

func (suite *ScriptRunSuite) TestPathProvidesLang() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	temp, err := os.MkdirTemp("", filepath.Base(t.Name()))
	require.NoError(t, err)

	tf := filepath.Join(temp, "python3")
	if runtime.GOOS == "windows" {
		tf = filepath.Join(temp, "python3.exe")
	}

	err = fileutils.Touch(tf)
	require.NoError(t, err)
	defer os.Remove(temp)

	require.NoError(t, os.Chmod(tf, 0770))

	exec := language.Python3.Executable().Filename()

	home, err := user.HomeDir()
	require.NoError(t, err)

	paths := []string{temp, home}
	pathStr := strings.Join(paths, string(os.PathListSeparator))

	assert.True(t, scriptrun.PathProvidesExec(filepath.Dir(tf), exec))
	assert.True(t, scriptrun.PathProvidesExec(pathStr, exec))
	assert.False(t, scriptrun.PathProvidesExec(pathStr, language.Unknown.String()))
	assert.False(t, scriptrun.PathProvidesExec("", exec))
}

func setupProjectWithScriptsExpectingArgs(t *testing.T, cmdName string) *projectfile.Project {
	if runtime.GOOS == "windows" {
		// Windows supports bash, but for the purpose of this test we only want to test cmd.exe, so ensure
		// that we run with cmd.exe even if the test is ran from bash
		os.Unsetenv("SHELL")
	} else {
		os.Setenv("SHELL", "bash")
	}

	tmpfile, err := os.CreateTemp("", "testRunCommand")
	require.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState/project"
scripts:
  - name: %s
    standalone: true
    value: |
      echo "ARGS|${1}|${2}|${3}|${4}|"`, cmdName)
	} else {
		contents = fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState/project"
scripts:
  - name: %s
    standalone: true
    language: batch
    value: |
      echo "ARGS|%%1|%%2|%%3|%%4|"`, cmdName)
	}
	err = yaml.Unmarshal([]byte(contents), project)

	require.Nil(t, err, "error unmarshalling project yaml")
	return project
}

func captureExecCommand(t *testing.T, tmplCmdName, cmdName string, cmdArgs []string) (string, error) {
	auth, err := authentication.LegacyGet()
	require.NoError(t, err)

	pjfile := setupProjectWithScriptsExpectingArgs(t, tmplCmdName)
	proj, err := project.New(pjfile, nil)
	require.NoError(t, err)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	outStr, outErr := osutil.CaptureStdout(func() {
		scriptRun := scriptrun.New(primer.New(auth, outputhelper.NewCatcher(), subshell.New(cfg), proj, cfg, blackhole.New()))
		var script *project.Script
		if script, err = proj.ScriptByName(cmdName); err == nil {
			err = scriptRun.Run(script, cmdArgs)
		}
	})
	require.NoError(t, outErr, "error capturing stdout")
	return outStr, err
}

func assertExecCommandProcessesArgs(t *testing.T, tmplCmdName, cmdName string, cmdArgs []string, expectedStdout string) {
	outStr, err := captureExecCommand(t, tmplCmdName, cmdName, cmdArgs)

	require.NoError(t, err, "unexpected error occurred")

	assert.Contains(t, outStr, expectedStdout)
}

func assertExecCommandFails(t *testing.T, tmplCmdName, cmdName string, cmdArgs []string) {
	_, err := captureExecCommand(t, tmplCmdName, cmdName, cmdArgs)
	require.Error(t, err, "run with error")
}

func (suite *ScriptRunSuite) TestArgs() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	t := suite.T()

	assertExecCommandFails(t, "junk", "", []string{})
	assertExecCommandFails(t, "junk", "--", []string{})
	assertExecCommandProcessesArgs(t, "foo", "foo", []string{"--"}, "ARGS|--||||")
	assertExecCommandProcessesArgs(t, "bar", "bar", []string{"baz", "bee"}, "ARGS|baz|bee|||")
	assertExecCommandFails(t, "junk", "--", []string{"foo", "geez"})
	assertExecCommandFails(t, "junk", "-f", []string{"--foo", "geez"})
	assertExecCommandProcessesArgs(t, "release", "release", []string{"--", "the", "kraken"}, "ARGS|--|the|kraken||")
	assertExecCommandProcessesArgs(t, "release", "release", []string{"the", "kraken"}, "ARGS|the|kraken|||")
	assertExecCommandProcessesArgs(t, "foo", "foo", []string{"bar", "--", "bees", "wax"}, "ARGS|bar|--|bees|wax|")
	assertExecCommandProcessesArgs(t, "foo", "foo", []string{"--bar", "--", "bees", "--wax"}, "ARGS|--bar|--|bees|--wax|")
}
