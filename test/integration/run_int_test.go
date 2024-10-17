package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
)

type RunIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RunIntegrationTestSuite) createProjectFile(ts *e2e.Session, name, commitID string) {
	root := environment.GetRootPathUnsafe()
	interruptScript := filepath.Join(root, "test", "integration", "assets", "run", "interrupt.go")
	err := fileutils.CopyFile(interruptScript, filepath.Join(ts.Dirs.Work, "interrupt.go"))
	suite.Require().NoError(err)

	configFileContent := strings.TrimPrefix(fmt.Sprintf(`
project: https://platform.activestate.com/%s
scripts:
  - name: test-interrupt
    description: A script that sleeps for a very long time.  It should be interrupted.  The first interrupt does not terminate.
    standalone: true
    language: bash
    value: |
        go build -o ./interrupt ./interrupt.go
        ./interrupt
    if: ne .OS.Name "Windows"
  - name: test-interrupt
    description: A script that sleeps for a very long time.  It should be interrupted.  The first interrupt does not terminate.
    standalone: true
    language: bash
    value: |
        go build -o .\interrupt.exe .\interrupt.go
        .\interrupt.exe
    if: eq .OS.Name "Windows"
  - name: helloWorld
    value: echo "Hello World!"
    standalone: true
    if: ne .OS.Name "Windows"
  - name: helloWorld
    standalone: true
    value: echo Hello World!
    if: eq .OS.Name "Windows"
  - name: testMultipleLanguages
    value: |
      import sys
      print(sys.version)
    language: python2,python3
  - name: nonZeroExit
    value: |
      exit 123
    standalone: true
    language: bash
`, name), "\n")

	ts.PrepareActiveStateYAML(configFileContent)
	ts.PrepareCommitIdFile(commitID)
}

func (suite *RunIntegrationTestSuite) SetupTest() {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("bash"); err != nil {
			suite.T().Skip("This test requires a bash shell in your PATH")
		}
	}
}

func (suite *RunIntegrationTestSuite) expectTerminateBatchJob(cp *e2e.SpawnedCmd) {
	if runtime.GOOS == "windows" {
		// send N to "Terminate batch job (Y/N)" question
		cp.Expect("Terminate batch job")
		time.Sleep(200 * time.Millisecond)
		cp.SendLine("N")
		cp.Expect("N", termtest.OptExpectTimeout(500*time.Millisecond))
	}
}

// TestActivatedEnv is a regression test for the following tickets:
// - https://www.pivotaltracker.com/story/show/167523128
// - https://www.pivotaltracker.com/story/show/169509213
func (suite *RunIntegrationTestSuite) TestInActivatedEnv() {
	suite.OnlyRunForTags(tagsuite.Run, tagsuite.Activate, tagsuite.Interrupt)
	if runtime.GOOS != "linux" && e2e.RunningOnCI() {
		suite.T().Skip("Windows CI does not support ctrl-c events, mac CI has Golang build issues")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.Spawn("activate")
	cp.Expect("Activated")
	cp.ExpectInput()

	// We're on Linux CI, so it's okay to use the OS's installed Python for this test.
	// It's costly to source our own for this test.
	cp.SendLine(fmt.Sprintf("%s run testMultipleLanguages", ts.Exe))
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Empty")
	cp.Expect("3")

	cp.SendLine(fmt.Sprintf("%s run test-interrupt", cp.Executable()))
	cp.Expect("Start of script", termtest.OptExpectTimeout(10*time.Second))
	cp.SendCtrlC()
	cp.Expect("received interrupt", termtest.OptExpectTimeout(5*time.Second))
	cp.Expect("After first sleep or interrupt", termtest.OptExpectTimeout(2*time.Second))
	cp.SendCtrlC()
	suite.expectTerminateBatchJob(cp)

	cp.SendLine("exit 0")
	cp.ExpectExitCode(0)
	suite.Require().NotContains(
		cp.Output(), "not printed after second interrupt",
	)
}

// tests that convenience commands for activestate.yaml scripts are available
// in bash subshells from the activated state
func (suite *RunIntegrationTestSuite) TestScriptBashSubshell() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("bash subshells are not supported by our tests on windows")
	}

	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.SpawnWithOpts(e2e.OptArgs("activate"), e2e.OptAppendEnv("SHELL=bash"))
	cp.Expect("Activated")
	cp.ExpectInput()

	cp.SendLine("helloWorld")
	cp.Expect("Hello World!")
	cp.SendLine("bash -c helloWorld")
	cp.Expect("Hello World!")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestOneInterrupt() {
	suite.OnlyRunForTags(tagsuite.Run, tagsuite.Interrupt, tagsuite.Critical)
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("Windows CI does not support ctrl-c events")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.Spawn("run", "test-interrupt")
	cp.Expect("Start of script")
	// interrupt the first (very long) sleep
	cp.SendCtrlC()

	cp.Expect("received interrupt", termtest.OptExpectTimeout(3*time.Second))
	cp.Expect("After first sleep or interrupt", termtest.OptExpectTimeout(2*time.Second))
	cp.Expect("After second sleep")
	suite.expectTerminateBatchJob(cp)
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestTwoInterrupts() {
	suite.OnlyRunForTags(tagsuite.Run, tagsuite.Interrupt)
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("Windows CI does not support ctrl-c events")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.Spawn("run", "test-interrupt")
	cp.Expect("Start of script")
	cp.SendCtrlC()
	cp.Expect("received interrupt", termtest.OptExpectTimeout(3*time.Second))
	cp.Expect("After first sleep or interrupt", termtest.OptExpectTimeout(2*time.Second))
	cp.SendCtrlC()
	suite.expectTerminateBatchJob(cp)
	cp.ExpectExitCode(123)
	ts.IgnoreLogErrors()
	suite.Require().NotContains(
		cp.Output(), "not printed after second interrupt",
	)
}

func (suite *RunIntegrationTestSuite) TestRun_Help() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.Spawn("run", "-h")
	cp.Expect("Usage")
	cp.Expect("Arguments")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_ExitCode() {
	suite.OnlyRunForTags(tagsuite.Run, tagsuite.ExitCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.Spawn("run", "nonZeroExit")
	cp.ExpectExitCode(123)
}

func (suite *RunIntegrationTestSuite) TestRun_Unauthenticated() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, "ActiveState-CLI/Python2", "fbc613d6-b0b1-4f84-b26e-4aa5869c4e54")

	cp := ts.Spawn("activate")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()

	cp.SendLine(fmt.Sprintf("%s run testMultipleLanguages", cp.Executable()))
	cp.Expect("2")
	cp.ExpectInput(termtest.OptExpectTimeout(120 * time.Second))

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_DeprecatedLackingLanguage() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	cp := ts.Spawn("run", "helloWorld")
	cp.Expect("Deprecation Warning", termtest.OptExpectTimeout(5*time.Second))
	cp.Expect("Hello", termtest.OptExpectTimeout(5*time.Second))
}

func (suite *RunIntegrationTestSuite) TestRun_BadLanguage() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	asyFilename := filepath.Join(ts.Dirs.Work, "activestate.yaml")
	asyFile, err := os.OpenFile(asyFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	suite.Require().NoError(err, "config is opened for appending")
	defer asyFile.Close()

	_, err = asyFile.WriteString(strings.TrimPrefix(`
  - name: badLanguage
    language: bax
    value: echo "shouldn't show"
`, "\n"))
	suite.Require().NoError(err, "extra config is appended")

	cp := ts.Spawn("run", "badLanguage")
	cp.Expect("The language for this script is not supported", termtest.OptExpectTimeout(5*time.Second))
}

func (suite *RunIntegrationTestSuite) TestRun_Perl_Variable() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("Testing exec of Perl with variables is not applicable on Windows")
	}

	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Perl-5.32", "a4762408-def6-41e4-b709-4cb548765005")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate"),
		e2e.OptAppendEnv("PERL_VERSION=does_not_exist"),
	)
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()

	cp.SendLine("perl -MEnglish -e 'print $PERL_VERSION'")
	cp.Expect("v5.32.0")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_Args() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	asyFilename := filepath.Join(ts.Dirs.Work, "activestate.yaml")
	asyFile, err := os.OpenFile(asyFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	suite.Require().NoError(err, "config is opened for appending")
	defer asyFile.Close()

	lang := project.DefaultScriptLanguage()[0].String()
	cmd := `if [ "$1" = "<3" ]; then echo heart; fi`
	if runtime.GOOS == "windows" {
		cmd = `@echo off
      if %1=="<3" (echo heart)` // need to match indent of YAML below
	}
	_, err = asyFile.WriteString(strings.TrimPrefix(fmt.Sprintf(`
  - name: args
    language: %s
    value: |
      %s
`, lang, cmd), "\n"))
	suite.Require().NoError(err, "extra config is appended")

	arg := "<3"
	cp := ts.Spawn("run", "args", arg)
	cp.Expect("heart", termtest.OptExpectTimeout(5*time.Second))
}

func TestRunIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RunIntegrationTestSuite))
}
