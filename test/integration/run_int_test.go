package integration

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type RunIntegrationTestSuite struct {
	suite.Suite
}

func (suite *RunIntegrationTestSuite) createProjectFile(ts *e2e.Session, pythonVersion int) {
	root := environment.GetRootPathUnsafe()
	interruptScript := filepath.Join(root, "test", "integration", "assets", "run", "interrupt.go")
	fileutils.CopyFile(interruptScript, filepath.Join(ts.Dirs.Work, "interrupt.go"))

	// ActiveState-CLI/Python3 is just a place-holder that is never used
	configFileContent := strings.TrimSpace(fmt.Sprintf(`
project: https://platform.activestate.com/ActiveState-CLI/Python%d?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
scripts:
  - name: test-interrupt
    description: A script that sleeps for a very long time.  It should be interrupted.  The first interrupt does not terminate.
    standalone: true
    value: |
        go build -o ./interrupt ./interrupt.go
        ./interrupt
    constraints:
        os: linux,macos
  - name: test-interrupt
    description: A script that sleeps for a very long time.  It should be interrupted.  The first interrupt does not terminate.
    standalone: true
    value: |
        go build -o .\interrupt.exe .\interrupt.go
        .\interrupt.exe
    constraints:
        os: windows
  - name: helloWorld
    value: echo "Hello World!"
    standalone: true
    constraints:
      os: linux,macos
  - name: helloWorld
    standalone: true
    value: echo Hello World!
    constraints:
    os: windows
  - name: helloWorldPython
    value: print("Hello Python!")
    language: python3
  - name: helloWorldMultiple
    value: |
      import sys
      print(sys.version)
    language: python2,python3
`, pythonVersion))

	ts.PrepareActiveStateYAML(configFileContent)
}

func (suite *RunIntegrationTestSuite) SetupTest() {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("bash"); err != nil {
			suite.T().Skip("This test requires a bash shell in your PATH")
		}
	}
}

func (suite *RunIntegrationTestSuite) TearDownTest() {
	projectfile.Reset()
}

func (suite *RunIntegrationTestSuite) expectTerminateBatchJob(cp *termtest.ConsoleProcess) {
	if runtime.GOOS == "windows" {
		// send N to "Terminate batch job (Y/N)" question
		cp.Expect("Terminate batch job")
		time.Sleep(200 * time.Millisecond)
		cp.SendLine("N")
		cp.Expect("N", 500*time.Millisecond)
	}
}

// TestActivatedEnv is a regression test for the following tickets:
// - https://www.pivotaltracker.com/story/show/167523128
// - https://www.pivotaltracker.com/story/show/169509213
func (suite *RunIntegrationTestSuite) TestInActivatedEnv() {
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("Windows CI does not support ctrl-c events")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, 3)
	ts.LoginAsPersistentUser()
	defer ts.LogoutUser()

	cp := ts.Spawn("activate")
	cp.Expect("Activating state: ActiveState-CLI/Python3")
	cp.WaitForInput(10 * time.Second)

	cp.SendLine(fmt.Sprintf("%s run test-interrupt", cp.Executable()))
	cp.Expect("Start of script", 5*time.Second)
	cp.SendCtrlC()
	cp.Expect("received interrupt", 3*time.Second)
	cp.Expect("After first sleep or interrupt", 2*time.Second)
	cp.SendCtrlC()
	suite.expectTerminateBatchJob(cp)

	cp.SendLine("exit 0")
	cp.ExpectExitCode(0)
	suite.Require().NotContains(
		cp.TrimmedSnapshot(), "not printed after second interrupt",
	)
}

func (suite *RunIntegrationTestSuite) TestOneInterrupt() {
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
	// interrupt the first (very long) sleep
	cp.SendCtrlC()

	cp.Expect("received interrupt", 3*time.Second)
	cp.Expect("After first sleep or interrupt", 2*time.Second)
	cp.Expect("After second sleep")
	suite.expectTerminateBatchJob(cp)
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestTwoInterrupts() {
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
	cp.Expect("received interrupt", 3*time.Second)
	cp.Expect("After first sleep or interrupt", 2*time.Second)
	cp.SendCtrlC()
	suite.expectTerminateBatchJob(cp)
	cp.ExpectExitCode(123)
	suite.Require().NotContains(
		cp.TrimmedSnapshot(), "not printed after second interrupt",
	)
}

func (suite *RunIntegrationTestSuite) TestRun_Help() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, 3)

	cp := ts.Spawn("run", "-h")
	cp.Expect("Usage")
	cp.Expect("Arguments")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_Unauthenticated() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, 3)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activating state: ActiveState-CLI/Python3")
	cp.WaitForInput(120 * time.Second)

	cp.SendLine(fmt.Sprintf("%s run helloWorldPython", cp.Executable()))
	cp.Expect("Hello Python!", 5*time.Second)

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_DeprecatedLackingLanguage() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.createProjectFile(ts, 3)

	cp := ts.Spawn("run", "helloWorld")
	cp.Expect("DEPRECATION", 5*time.Second)
	cp.Expect("Hello", 5*time.Second)
}

func (suite *RunIntegrationTestSuite) TestRun_MultipleLanguagesPython2() {
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("Windows CI does not support ctrl-c events")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, 2)

	ts.LoginAsPersistentUser()
	defer ts.LogoutUser()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activating state: ActiveState-CLI/Python2")
	cp.WaitForInput(120 * time.Second)

	cp.SendLine(fmt.Sprintf("%s run helloWorldMultiple", cp.Executable()))
	cp.Expect("2")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *RunIntegrationTestSuite) TestRun_MultipleLanguagesPython3() {
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("Windows CI does not support ctrl-c events")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.createProjectFile(ts, 3)

	ts.LoginAsPersistentUser()
	defer ts.LogoutUser()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activating state: ActiveState-CLI/Python3")
	cp.WaitForInput(120 * time.Second)

	cp.SendLine(fmt.Sprintf("%s run helloWorldMultiple", cp.Executable()))
	cp.Expect("3")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func TestRunIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RunIntegrationTestSuite))
}
