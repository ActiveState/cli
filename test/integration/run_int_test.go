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

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type RunIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RunIntegrationTestSuite) createProjectFile(ts *e2e.Session, pythonVersion int) {
	root := environment.GetRootPathUnsafe()
	interruptScript := filepath.Join(root, "test", "integration", "assets", "run", "interrupt.go")
	fileutils.CopyFile(interruptScript, filepath.Join(ts.Dirs.Work, "interrupt.go"))

	// ActiveState-CLI/Python3 is just a place-holder that is never used
	configFileContent := strings.TrimPrefix(fmt.Sprintf(`
project: https://platform.activestate.com/ActiveState-CLI/Python%d
scripts:
  - name: test-interrupt
    description: A script that sleeps for a very long time.  It should be interrupted.  The first interrupt does not terminate.
    standalone: true
    value: |
        go build -o ./interrupt ./interrupt.go
        ./interrupt
    if: ne .OS.Name "Windows"
  - name: test-interrupt
    description: A script that sleeps for a very long time.  It should be interrupted.  The first interrupt does not terminate.
    standalone: true
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
`, pythonVersion), "\n")

	ts.PrepareActiveStateYAML(configFileContent)
	ts.PrepareCommitIdFile("fbc613d6-b0b1-4f84-b26e-4aa5869c4e54")
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

	suite.createProjectFile(ts, 3)

	cp := ts.Spawn("activate")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput(termtest.OptExpectTimeout(10 * time.Second))

	cp.SendLine(fmt.Sprintf("%s run testMultipleLanguages", ts.Exe))
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Python3")
	cp.Expect("3")

	cp.SendLine(fmt.Sprintf("%s run test-interrupt", cp.Executable()))
	cp.Expect("Start of script", termtest.OptExpectTimeout(5*time.Second))
	cp.SendCtrlC()
	cp.Expect("received interrupt", termtest.OptExpectTimeout(3*time.Second))
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

	suite.createProjectFile(ts, 3)

	cp := ts.SpawnWithOpts(e2e.OptArgs("activate"), e2e.OptAppendEnv("SHELL=bash"))
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput(termtest.OptExpectTimeout(10 * time.Second))

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
	suite.createProjectFile(ts, 3)

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
	suite.createProjectFile(ts, 3)

	ts.LoginAsPersistentUser()
	defer ts.LogoutUser()

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
	suite.createProjectFile(ts, 3)

	cp := ts.Spawn("run", "-h")
	cp.Expect("Usage")
	cp.Expect("Arguments")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_ExitCode() {
	suite.OnlyRunForTags(tagsuite.Run, tagsuite.ExitCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, 3)

	cp := ts.Spawn("run", "nonZeroExit")
	cp.ExpectExitCode(123)
}

func (suite *RunIntegrationTestSuite) TestRun_Unauthenticated() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, 2)

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput(termtest.OptExpectTimeout(10 * time.Second))

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

	suite.createProjectFile(ts, 3)

	cp := ts.Spawn("run", "helloWorld")
	cp.Expect("Deprecation Warning", termtest.OptExpectTimeout(5*time.Second))
	cp.Expect("Hello", termtest.OptExpectTimeout(5*time.Second))
}

func (suite *RunIntegrationTestSuite) TestRun_BadLanguage() {
	suite.OnlyRunForTags(tagsuite.Run)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, 3)

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
		e2e.OptAppendEnv(
			constants.DisableRuntime+"=false",
			"PERL_VERSION=does_not_exist",
		),
	)
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput(termtest.OptExpectTimeout(10 * time.Second))

	cp.SendLine("perl -MEnglish -e 'print $PERL_VERSION'")
	cp.Expect("v5.32.0")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func TestRunIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RunIntegrationTestSuite))
}
