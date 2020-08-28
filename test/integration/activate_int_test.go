package integration

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

type ActivateIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3() {
	suite.activatePython("3")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3_zsh() {
	if _, err := exec.LookPath("zsh"); err != nil {
		suite.T().Skip("This test requires a zsh shell in your PATH")
	}
	suite.activatePython("3", "SHELL=zsh")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython2() {
	suite.activatePython("2")
}

func (suite *ActivateIntegrationTestSuite) TestActivateWithoutRuntime() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python3")
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect("activated state", 20*time.Second)
	cp.WaitForInput(10 * time.Second)

	cp.SendLine("exit 123")
	cp.ExpectExitCode(123, 10*time.Second)
}

func (suite *ActivateIntegrationTestSuite) TestActivateNotOnPath() {
	ts := e2e.NewNoPathUpdate(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python3")
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect("activated state", 20*time.Second)
	cp.WaitForInput(10 * time.Second)

	if runtime.GOOS == "windows" {
		cp.SendLine("doskey /macros | findstr state=")
	} else {
		cp.SendLine("alias state")
	}
	cp.Expect("state=")

	cp.SendLine("state --version")
	cp.Expect("ActiveState")

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

// TestActivatePythonByHostOnly Tests whether we are only pulling in the build for the target host
func (suite *ActivateIntegrationTestSuite) TestActivatePythonByHostOnly() {
	if runtime.GOOS != "linux" {
		suite.T().Skip("not currently testing this OS")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	projectName := "Python-LinuxWorks"
	cp := ts.Spawn("activate", "cli-integration-tests/"+projectName, "--path="+ts.Dirs.Work)

	cp.Expect("Activating state")
	cp.Expect("activated state", 120*time.Second)
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) assertCompletedStatusBarReport(snapshot string) {
	// ensure that terminal contains output "Installing x/y" with x, y numbers and x=y
	installingString := regexp.MustCompile(
		"Installing *([0-9]+) */ *([0-9]+)",
	).FindAllStringSubmatch(snapshot, -1)
	suite.Require().Greater(len(installingString), 0, "no match for Installing x / x in\n%s", snapshot)
	le := len(installingString) - 1
	suite.Require().Equalf(
		installingString[le][1], installingString[le][2],
		"expected all artifacts are reported to be installed, got %s in\n%s", installingString[0][0], snapshot,
	)
}

func (suite *ActivateIntegrationTestSuite) activatePython(version string, extraEnv ...string) {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Python"+version),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		e2e.AppendEnv(extraEnv...),
	)
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect("Downloading", 20*time.Second)
	cp.Expect("Installing", 120*time.Second)
	cp.Expect("activated state", 120*time.Second)

	suite.assertCompletedStatusBarReport(cp.Snapshot())

	// ensure that shell is functional
	cp.WaitForInput()

	pythonExe := "python" + version

	cp.SendLine(pythonExe + " -c \"import sys; print(sys.copyright)\"")
	cp.Expect("ActiveState Software Inc.")

	cp.SendLine(pythonExe + " -c \"import pytest; print(pytest.__doc__)\"")
	cp.Expect("unit and functional testing")

	// test that other executables that use python work as well
	pipExe := "pip" + version
	cp.SendLine(fmt.Sprintf("%s --version", pipExe))
	pipVersionRe := regexp.MustCompile(`pip \d+(?:\.\d+)+ from ([^ ]+) \(python`)
	cp.ExpectRe(pipVersionRe.String())
	pipVersionMatch := pipVersionRe.FindStringSubmatch(cp.TrimmedSnapshot())
	suite.Require().Len(pipVersionMatch, 2, "expected pip version to match")
	suite.Contains(pipVersionMatch[1], "cache", "pip loaded from activestate cache dir")

	// de-activate shell
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3_Forward() {
	var project string
	if runtime.GOOS == "darwin" {
		project = "Activate-MacOS"
	} else {
		project = "Python3"
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	contents := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/%s"
branch: %s
version: %s
`, project, constants.BranchName, constants.Version))

	ts.PrepareActiveStateYAML(contents)

	// Ensure we have the most up to date version of the project before activating
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("pull"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Your activestate.yaml has been updated to the latest version available")
	cp.Expect("If you have any active instances of this project open in other terminals")
	cp.ExpectExitCode(0)

	c2 := ts.Spawn("activate")
	c2.Expect(fmt.Sprintf("Activating state: ActiveState-CLI/%s", project))

	// not waiting for activation, as we test that part in a different test
	c2.WaitForInput()
	c2.SendLine("exit")
	c2.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivatePerl() {
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Perl not supported on macOS")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Perl"),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
		),
	)
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect("Downloading", 20*time.Second)
	cp.Expect("Installing", 120*time.Second)
	cp.Expect("activated state", 120*time.Second)

	suite.assertCompletedStatusBarReport(cp.Snapshot())

	// ensure that shell is functional
	cp.WaitForInput()

	cp.SendLine("perldoc -l DBD::Pg")
	// Expect the source code to be installed in the cache directory
	// Note: At least for Windows we cannot expect cp.Dirs.Cache, because it is unreliable how the path name formats are unreliable (sometimes DOS 8.3 format, sometimes not)
	cp.Expect("cache")
	cp.Expect("Pg.pm")

	// Expect PPM shim to be installed
	cp.SendLine("ppm")
	cp.Expect("Your command is being forwarded to `state packages`.")

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_Subdir() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	fail := fileutils.Mkdir(ts.Dirs.Work, "foo", "bar", "baz")
	suite.Require().NoError(fail.ToError())

	// Create the project file at the root of the temp dir
	content := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
branch: %s
version: %s
`, constants.BranchName, constants.Version))

	ts.PrepareActiveStateYAML(content)

	// Pull to ensure we have an up to date config file
	cp := ts.Spawn("pull")
	cp.Expect("Your activestate.yaml has been updated to the latest version available")
	cp.Expect("If you have any active instances of this project open in other terminals")
	cp.ExpectExitCode(0)

	// Activate in the subdirectory
	c2 := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.WithWorkDirectory(filepath.Join(ts.Dirs.Work, "foo", "bar", "baz")),
	)
	c2.Expect("Activating state: ActiveState-CLI/Python3")

	c2.WaitForInput()
	c2.SendLine("exit")
	c2.ExpectExitCode(0)

}

func (suite *ActivateIntegrationTestSuite) TestInit_Activation_NoCommitID() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("init", namespace, "python3")
	cp.Expect(fmt.Sprintf("Project '%s' has been successfully initialized", namespace))
	cp.ExpectExitCode(0)
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectLongString(locale.Tr("err_project_no_commit", url))
	cp.ExpectExitCode(1)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_InterruptedInstallation() {
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("interrupting installation does not work on Windows on CI")
	}
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cp := ts.Spawn("deploy", "install", "ActiveState-CLI/small-python")
	cp.Expect("Downloading")
	cp.Expect("Installing")
	// interrupting installation
	cp.SendCtrlC()
	cp.ExpectNotExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_FromCache() {
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Downloading")
	cp.Expect("Installing")
	cp.Expect("activated state")

	suite.assertCompletedStatusBarReport(cp.Snapshot())
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// next activation is cached
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Downloading required artifacts")
}

func (suite *ActivateIntegrationTestSuite) TestActivate_JSON() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python3", "--output", "json")
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect(`"ACTIVESTATE_ACTIVATED":"`)
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_Command() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python3", "-c", "echo CUSTOM_COMMAND")
	cp.Expect("Where would you like to checkout")
	cp.SendLine(cp.WorkDirectory())
	cp.Expect("CUSTOM_COMMAND")
	cp.ExpectExitCode(0)
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ActivateIntegrationTestSuite))
}
