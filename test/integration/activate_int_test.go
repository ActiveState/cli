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
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ActivateIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3() {
	suite.OnlyRunForTags(tagsuite.Python, tagsuite.Activate, tagsuite.Critical)
	suite.activatePython("3")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3_zsh() {
	suite.OnlyRunForTags(tagsuite.Python, tagsuite.Activate, tagsuite.Shell)
	if _, err := exec.LookPath("zsh"); err != nil {
		suite.T().Skip("This test requires a zsh shell in your PATH")
	}
	suite.activatePython("3", "SHELL=zsh")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython2() {
	suite.OnlyRunForTags(tagsuite.Python, tagsuite.Activate)
	suite.activatePython("2")
}

func (suite *ActivateIntegrationTestSuite) TestActivateWithoutRuntime() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Activate, tagsuite.ExitCode)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python2")
	cp.Expect("Activated")
	cp.WaitForInput()

	cp.SendLine("exit 123")
	cp.ExpectExitCode(123)
}

func (suite *ActivateIntegrationTestSuite) TestActivateUsingCommitID() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Python3#6d9280e7-75eb-401a-9e71-0d99759fbad3", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated", 40*time.Second)
	cp.WaitForInput(10 * time.Second)

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivateNotOnPath() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Activate)
	ts := e2e.NewNoPathUpdate(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "activestate-cli/small-python", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activated", 40*time.Second)
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
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	projectName := "Python-LinuxWorks"
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "cli-integration-tests/"+projectName, "--path="+ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	if runtime.GOOS == "linux" {
		cp.Expect("Creating a Virtual Environment")
		cp.Expect("Activated")
		cp.WaitForInput(40 * time.Second)
		cp.SendLine("exit")
		cp.ExpectExitCode(0)
	} else if runtime.GOOS == "windows" {
		// We can definitely improve this error, but this particular test is testing that we can still activate on the
		// platform that DOES match (ie. Linux)
		cp.Expect("Could not update runtime installation")
		cp.ExpectNotExitCode(0)
	} else {
		cp.Expect("Your current platform")
		cp.Expect("does not appear to be configured")
		cp.ExpectNotExitCode(0)
	}
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

	namespace := "ActiveState-CLI/Python" + version

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", namespace),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		e2e.AppendEnv(extraEnv...),
	)

	cp.Expect("Activated")
	// ensure that shell is functional
	cp.WaitForInput()

	pythonExe := "python" + version

	cp.SendLine(pythonExe + " -c \"import sys; print(sys.copyright)\"")
	cp.Expect("ActiveState Software Inc.")

	if runtime.GOOS == "windows" {
		cp.SendLine("where " + pythonExe)
		cp.Expect(`\exec\` + pythonExe)
	} else {
		cp.SendLine("which " + pythonExe)
		cp.Expect("/exec/" + pythonExe)
	}

	cp.SendLine(pythonExe + " -c \"import pytest; print(pytest.__doc__)\"")
	cp.Expect("unit and functional testing")

	cp.SendLine("state activate --default something/else")
	cp.ExpectLongString("Cannot set something/else as the global default project while in an activated state")

	cp.SendLine("state activate --default")
	cp.ExpectLongString("Creating a Virtual Environment")
	cp.WaitForInput(40 * time.Second)
	pythonShim := pythonExe
	if runtime.GOOS == "windows" {
		pythonShim = pythonExe + ".bat"
	}

	// test that other executables that use python work as well
	pipExe := "pip" + version
	cp.SendLine(fmt.Sprintf("%s --version", pipExe))
	pipVersionRe := regexp.MustCompile(`pip \d+(?:\.\d+)+ from ([^ ]+) \(python`)
	cp.ExpectRe(pipVersionRe.String())
	pipVersionMatch := pipVersionRe.FindStringSubmatch(cp.TrimmedSnapshot())
	suite.Require().Len(pipVersionMatch, 2, "expected pip version to match")
	suite.Contains(pipVersionMatch[1], "cache", "pip loaded from activestate cache dir")

	executor := filepath.Join(ts.Dirs.DefaultBin, pythonShim)
	// check that default activation works
	cp = ts.SpawnCmdWithOpts(
		executor,
		e2e.WithArgs("-c", "import sys; print(sys.copyright);"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("ActiveState Software Inc.")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_PythonPath() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", namespace),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated")
	// ensure that shell is functional
	cp.WaitForInput()

	// test that PYTHONPATH is preserved in environment (https://www.pivotaltracker.com/story/show/178458102)
	if runtime.GOOS == "windows" {
		cp.Send("set PYTHONPATH=/custom_pythonpath")
		cp.SendLine(`python3 -c 'import os; print(os.environ["PYTHONPATH"]);'`)
	} else {
		cp.SendLine(`PYTHONPATH=/custom_pythonpath python3 -c 'import os; print(os.environ["PYTHONPATH"]);'`)
	}
	cp.Expect("/custom_pythonpath")

	// de-activate shell
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ExecIntegrationTestSuite) TestActivate_SpaceInCacheDir() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cacheDir := filepath.Join(ts.Dirs.Cache, "dir with spaces")
	err := fileutils.MkdirUnlessExists(cacheDir)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.CacheEnvVarName, cacheDir)),
		e2e.AppendEnv(fmt.Sprintf(`%s=""`, constants.DisableRuntime)),
		e2e.WithArgs("activate", "ActiveState-CLI/Python3"),
	)

	cp.SendLine("python3 --version")
	cp.Expect("Python 3.")

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivatePerl() {
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.Perl)
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

	cp.Expect("Downloading", 40*time.Second)
	cp.Expect("Installing", 140*time.Second)
	cp.Expect("Activated")

	suite.assertCompletedStatusBarReport(cp.Snapshot())

	// ensure that shell is functional
	cp.WaitForInput()

	cp.SendLine("perldoc -l DBI::DBD")
	// Expect the source code to be installed in the cache directory
	// Note: At least for Windows we cannot expect cp.Dirs.Cache, because it is unreliable how the path name formats are unreliable (sometimes DOS 8.3 format, sometimes not)
	cp.Expect("cache")
	cp.Expect("DBD.pm")

	// Currently CI is searching for PPM in the @INC first before attempting
	// to execute a script. https://activestatef.atlassian.net/browse/DX-620
	if runtime.GOOS != "windows" {
		// Expect PPM shim to be installed
		cp.SendLine("ppm list")
		cp.Expect("Shimming command")
	}

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_Subdir() {
	suite.OnlyRunForTags(tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	err := fileutils.Mkdir(ts.Dirs.Work, "foo", "bar", "baz")
	suite.Require().NoError(err)

	// Create the project file at the root of the temp dir
	content := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
branch: %s
version: %s
`, constants.BranchName, constants.Version))

	ts.PrepareActiveStateYAML(content)

	// Pull to ensure we have an up to date config file
	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated to")
	cp.ExpectExitCode(0)

	// Activate in the subdirectory
	c2 := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.WithWorkDirectory(filepath.Join(ts.Dirs.Work, "foo", "bar", "baz")),
	)
	c2.Expect("Activated")

	c2.WaitForInput(40 * time.Second)
	c2.SendLine("exit")
	c2.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_NamespaceWins() {
	suite.OnlyRunForTags(tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	identifyPath := "identifyable-path"
	targetPath := filepath.Join(ts.Dirs.Work, "foo", "bar", identifyPath)
	defer ts.Close()
	err := fileutils.Mkdir(targetPath)
	suite.Require().NoError(err)

	// Create the project file at the root of the temp dir
	content := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
`))

	ts.PrepareActiveStateYAML(content)

	// Pull to ensure we have an up to date config file
	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated to")
	cp.ExpectExitCode(0)

	// Activate in the subdirectory
	c2 := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Python2"), // activate a different namespace
		e2e.WithWorkDirectory(targetPath),
	)
	c2.ExpectLongString("ActiveState-CLI/Python2")
	c2.Expect("Activated")

	c2.WaitForInput(40 * time.Second)
	if runtime.GOOS == "windows" {
		c2.SendLine("@echo %cd%")
	} else {
		c2.SendLine("pwd")
	}
	c2.Expect(identifyPath)
	c2.SendLine("exit")
	c2.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_InterruptedInstallation() {
	suite.OnlyRunForTags(tagsuite.Activate)
	if runtime.GOOS == "windows" && e2e.RunningOnCI() {
		suite.T().Skip("interrupting installation does not work on Windows on CI")
	}
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cp := ts.Spawn("deploy", "install", "ActiveState-CLI/small-python")
	// interrupting installation
	cp.SendCtrlC()
	cp.ExpectNotExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_FromCache() {
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	err := ts.ClearCache()
	suite.Require().NoError(err)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Downloading")
	cp.Expect("Installing")
	cp.Expect("Activated")

	suite.assertCompletedStatusBarReport(cp.Snapshot())
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// next activation is cached
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Downloading")
}

func (suite *ActivateIntegrationTestSuite) TestActivate_JSON() {
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.Output)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python", "--output", "json", "--path", ts.Dirs.Work),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect(`"ACTIVESTATE_ACTIVATED":"`)
	cp.ExpectExitCode(0)
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ActivateIntegrationTestSuite))
}

func (suite *ActivateIntegrationTestSuite) TestActivateCommitURL() {
	suite.OnlyRunForTags(tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// https://platform.activestate.com/ActiveState-CLI/Python3/customize?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
	commitID := "fbc613d6-b0b1-4f84-b26e-4aa5869c4e54"
	contents := fmt.Sprintf("project: https://platform.activestate.com/commit/%s\n", commitID)
	ts.PrepareActiveStateYAML(contents)

	// Ensure we have the most up to date version of the project before activating
	cp := ts.Spawn("activate")
	cp.Expect("Activated")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_AlreadyActive() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", namespace),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated")
	// ensure that shell is functional
	cp.WaitForInput()

	cp.SendLine("state activate")
	cp.Expect("Your project is already active")
	cp.WaitForInput()
}

func (suite *ActivateIntegrationTestSuite) TestActivate_AlreadyActive_SameNamespace() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", namespace),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated")
	// ensure that shell is functional
	cp.WaitForInput()

	cp.SendLine(fmt.Sprintf("state activate %s", namespace))
	cp.Expect("Your project is already active")
	cp.WaitForInput()
}

func (suite *ActivateIntegrationTestSuite) TestActivate_AlreadyActive_DifferentNamespace() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", namespace),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated")
	// ensure that shell is functional
	cp.WaitForInput()

	cp.SendLine(fmt.Sprintf("state activate %s", "ActiveState-CLI/Perl-5.32"))
	cp.Expect("You cannot activate a new project when you are already in an activated state")
	cp.WaitForInput()
}
