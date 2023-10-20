package integration

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/rtutils"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
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
	close := suite.addForegroundSvc(ts)
	defer close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Python2")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	cp.ExpectInput()

	cp.SendLine("exit 123")
	cp.ExpectExitCode(123)
}

// addForegroundSvc launches the state-svc in a way where we can track its output for debugging purposes
// without this we are mostly blind to the svc exiting prematurely
func (suite *ActivateIntegrationTestSuite) addForegroundSvc(ts *e2e.Session) func() {
	cmd, stdout, stderr, err := exeutils.ExecuteInBackground(ts.SvcExe, []string{"foreground"}, func(cmd *exec.Cmd) error {
		cmd.Env = append(ts.Env, "VERBOSE=true", "") // For whatever reason the last entry is ignored..
		return nil
	})
	suite.Require().NoError(err)

	// Wait for the svc to be ready
	rtutils.Timeout(func() error {
		code := -1
		for code != 0 {
			code, _, _ = exeutils.Execute(ts.SvcExe, []string{"status"}, func(cmd *exec.Cmd) error {
				cmd.Env = ts.Env
				return nil
			})
		}
		return nil
	}, 10*time.Second)

	// Stop function
	return func() {
		go func() {
			defer func() {
				suite.Require().Nil(recover())
			}()
			stdout, stderr, err := exeutils.ExecSimple(ts.SvcExe, []string{"stop"}, ts.Env)
			suite.Require().NoError(err, "svc stop failed: %s\n%s", stdout, stderr)
		}()

		verifyExit := true

		err2 := rtutils.Timeout(func() error { return cmd.Wait() }, 10*time.Second)
		if err2 != nil {
			if !errors.Is(err2, rtutils.ErrTimeout) {
				suite.Require().NoError(err2)
			}
			suite.T().Logf("svc did not stop in time, Stdout:\n%s\n\nStderr:\n%s", stdout.String(), stderr.String())
			cmd.Process.Kill()
		}

		errMsg := fmt.Sprintf("svc foreground did not complete as expected. Stdout:\n%s\n\nStderr:\n%s", stdout.String(), stderr.String())
		if verifyExit {
			suite.Require().NoError(err2, errMsg)
			if cmd.ProcessState.ExitCode() != 0 {
				suite.FailNow(errMsg)
			}
		}

		// Goroutines don't necessarily cause the process to exit non-zero, so check for common errors/panics
		rx := regexp.MustCompile(`(?:runtime error|invalid memory address|nil pointer|goroutine)`)
		if rx.Match(stderr.Bytes()) {
			suite.FailNow(errMsg)
		}
	}
}

func (suite *ActivateIntegrationTestSuite) TestActivateUsingCommitID() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/Python3#6d9280e7-75eb-401a-9e71-0d99759fbad3", "--path", ts.Dirs.Work),
	)
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	cp.ExpectInput(termtest.OptExpectTimeout(10 * time.Second))

	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivateNotOnPath() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Activate)
	ts := e2e.NewNoPathUpdate(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "activestate-cli/small-python", "--path", ts.Dirs.Work),
	)
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	cp.ExpectInput(termtest.OptExpectTimeout(10 * time.Second))

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
	close := suite.addForegroundSvc(ts)
	defer close()

	projectName := "Python-LinuxWorks"
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "cli-integration-tests/"+projectName, "--path="+ts.Dirs.Work),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	if runtime.GOOS == "linux" {
		cp.Expect("Creating a Virtual Environment")
		cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
		cp.ExpectInput(termtest.OptExpectTimeout(40 * time.Second))
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

		if strings.Count(cp.Snapshot(), " x ") != 1 {
			suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
		}
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
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Python" + version

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", namespace),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		e2e.OptAppendEnv(extraEnv...),
	)

	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	// ensure that shell is functional
	cp.ExpectInput()

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

	cp.SendLine("state activate --default ActiveState-CLI/cli")
	cp.Expect("Cannot make ActiveState-CLI/cli always available for use while in an activated state")

	cp.SendLine("state activate --default")
	cp.Expect("Creating a Virtual Environment")
	cp.ExpectInput(termtest.OptExpectTimeout(40 * time.Second))
	pythonShim := pythonExe + exeutils.Extension

	// test that other executables that use python work as well
	pipExe := "pip" + version
	cp.SendLine(fmt.Sprintf("%s --version", pipExe))

	// Exit activated state
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	pendingOutput := cp.PendingOutput() // Without waiting for exit this isn't guaranteed to have our output yet

	// Assert pip output
	pipVersionRe := regexp.MustCompile(`pip \d+(?:\.\d+)+ from ([^ ]+) \(python`)
	pipVersionMatch := pipVersionRe.FindStringSubmatch(pendingOutput)
	suite.Require().Len(pipVersionMatch, 2, "expected pip version to match, pending output: %s", pendingOutput)
	suite.Contains(pipVersionMatch[1], "cache", "pip loaded from activestate cache dir")

	executor := filepath.Join(ts.Dirs.DefaultBin, pythonShim)
	// check that default activation works
	cp = ts.SpawnCmdWithOpts(
		executor,
		e2e.OptArgs("-c", "import sys; print(sys.copyright);"),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("ActiveState Software Inc.", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_PythonPath() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", namespace),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	// ensure that shell is functional
	cp.ExpectInput()

	// Verify that PYTHONPATH is set correctly to the installed site-packages, not a temp runtime
	// setup directory.
	if runtime.GOOS == "windows" {
		cp.SendLine("echo %PYTHONPATH%")
	} else {
		cp.SendLine("echo $PYTHONPATH")
	}
	suite.Assert().NotContains(cp.Output(), constants.LocalRuntimeTempDirectory)
	// Verify the temp runtime setup directory has been removed.
	runtimeFound := false
	entries, err := fileutils.ListDir(ts.Dirs.Cache, true)
	suite.Require().NoError(err)
	for _, entry := range entries {
		if entry.IsDir() && fileutils.DirExists(filepath.Join(entry.Path(), constants.LocalRuntimeEnvironmentDirectory)) {
			runtimeFound = true
			suite.Assert().NoDirExists(filepath.Join(entry.Path(), constants.LocalRuntimeTempDirectory))
		}
	}
	suite.Assert().True(runtimeFound, "runtime directory was not found in ts.Dirs.Cache")

	// test that PYTHONPATH is preserved in environment (https://www.pivotaltracker.com/story/show/178458102)
	if runtime.GOOS == "windows" {
		cp.SendLine("set PYTHONPATH=/custom_pythonpath")
		cp.SendLine(`python3 -c "import os; print(os.environ['PYTHONPATH']);"`)
	} else {
		cp.SendLine(`PYTHONPATH=/custom_pythonpath python3 -c 'import os; print(os.environ["PYTHONPATH"]);'`)
	}
	cp.Expect("/custom_pythonpath")

	// de-activate shell
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_SpaceInCacheDir() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	cacheDir := filepath.Join(ts.Dirs.Cache, "dir with spaces")
	err := fileutils.MkdirUnlessExists(cacheDir)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.CacheEnvVarName, cacheDir)),
		e2e.OptAppendEnv(fmt.Sprintf(`%s=""`, constants.DisableRuntime)),
		e2e.OptArgs("activate", "ActiveState-CLI/Python3"),
	)

	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
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
	close := suite.addForegroundSvc(ts)
	defer close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/Perl"),
		e2e.OptAppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
		),
	)

	cp.Expect("Downloading", termtest.OptExpectTimeout(40*time.Second))
	cp.Expect("Installing", termtest.OptExpectTimeout(140*time.Second))
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)

	suite.assertCompletedStatusBarReport(cp.Output())

	// ensure that shell is functional
	cp.ExpectInput()

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
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()
	err := fileutils.Mkdir(ts.Dirs.Work, "foo", "bar", "baz")
	suite.Require().NoError(err)

	// Create the project file at the root of the temp dir
	content := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
branch: %s
version: %s
`, constants.BranchName, constants.Version))

	ts.PrepareActiveStateYAML(content)
	ts.PrepareCommitIdFile("59404293-e5a9-4fd0-8843-77cd4761b5b5")

	// Pull to ensure we have an up to date config file
	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated to")
	cp.ExpectExitCode(0)

	// Activate in the subdirectory
	c2 := ts.SpawnWithOpts(
		e2e.OptArgs("activate"),
		e2e.OptWD(filepath.Join(ts.Dirs.Work, "foo", "bar", "baz")),
	)
	c2.Expect("Activated")

	c2.ExpectInput(termtest.OptExpectTimeout(40 * time.Second))
	c2.SendLine("exit")
	c2.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_NamespaceWins() {
	suite.OnlyRunForTags(tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	identifyPath := "identifyable-path"
	targetPath := filepath.Join(ts.Dirs.Work, "foo", "bar", identifyPath)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()
	err := fileutils.Mkdir(targetPath)
	suite.Require().NoError(err)

	// Create the project file at the root of the temp dir
	ts.PrepareProject("ActiveState-CLI/Python3", "")

	// Pull to ensure we have an up to date config file
	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated to")
	cp.ExpectExitCode(0)

	// Activate in the subdirectory
	c2 := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/Python2"), // activate a different namespace
		e2e.OptWD(targetPath),
		e2e.OptAppendEnv(constants.DisableLanguageTemplates+"=true"),
	)
	c2.Expect("ActiveState-CLI/Python2")
	c2.Expect("Activated")

	c2.ExpectInput(termtest.OptExpectTimeout(40 * time.Second))
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
	close := suite.addForegroundSvc(ts)
	defer close()

	cp := ts.SpawnShellWithOpts("bash", e2e.OptAppendEnv(constants.DisableRuntime+"=false"))
	cp.SendLine("state deploy install ActiveState-CLI/small-python")
	cp.Expect("Installing Runtime") // Ensure we don't send Ctrl+C too soon
	cp.SendCtrlC()
	cp.Expect("User interrupted")
	cp.SendLine("exit")
	cp.ExpectExit()
}

func (suite *ActivateIntegrationTestSuite) TestActivate_FromCache() {
	suite.OnlyRunForTags(tagsuite.Activate, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	err := ts.ClearCache()
	suite.Require().NoError(err)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Downloading")
	cp.Expect("Installing")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)

	suite.assertCompletedStatusBarReport(cp.Output())
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// next activation is cached
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.ExpectInput(e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
	suite.NotContains(cp.Output(), "Downloading")
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ActivateIntegrationTestSuite))
}

func (suite *ActivateIntegrationTestSuite) TestActivateCommitURL() {
	suite.OnlyRunForTags(tagsuite.Activate)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	// https://platform.activestate.com/ActiveState-CLI/Python3/customize?commitID=fbc613d6-b0b1-4f84-b26e-4aa5869c4e54
	commitID := "fbc613d6-b0b1-4f84-b26e-4aa5869c4e54"
	contents := fmt.Sprintf("project: https://platform.activestate.com/commit/%s\n", commitID)
	ts.PrepareActiveStateYAML(contents)

	// Ensure we have the most up to date version of the project before activating
	cp := ts.Spawn("activate", "--non-interactive") // do not prompt to migrate
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_AlreadyActive() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(e2e.OptArgs("activate", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	// ensure that shell is functional
	cp.ExpectInput()

	cp.SendLine("state activate")
	cp.Expect("Your project is already active")
	cp.ExpectInput()
}

func (suite *ActivateIntegrationTestSuite) TestActivate_AlreadyActive_SameNamespace() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(e2e.OptArgs("activate", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	// ensure that shell is functional
	cp.ExpectInput()

	cp.SendLine(fmt.Sprintf("state activate %s", namespace))
	cp.Expect("Your project is already active")
	cp.ExpectInput()
}

func (suite *ActivateIntegrationTestSuite) TestActivate_AlreadyActive_DifferentNamespace() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(e2e.OptArgs("activate", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	// ensure that shell is functional
	cp.ExpectInput()

	cp.SendLine(fmt.Sprintf("state activate %s", "ActiveState-CLI/Perl-5.32"))
	cp.Expect("You cannot activate a new project when you are already in an activated state")
	cp.ExpectInput()
}

func (suite *ActivateIntegrationTestSuite) TestActivateBranch() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Branches"

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", namespace, "--branch", "firstbranch"),
	)
	cp.Expect("Skipping runtime setup")
	cp.Expect("Activated")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivateBranchNonExistant() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Branches"

	cp := ts.SpawnWithOpts(e2e.OptArgs("activate", namespace, "--branch", "does-not-exist"))

	cp.Expect("has no branch")
}

func (suite *ActivateIntegrationTestSuite) TestActivateArtifactsCached() {
	suite.OnlyRunForTags(tagsuite.Activate)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	close := suite.addForegroundSvc(ts)
	defer close()

	namespace := "ActiveState-CLI/Python3"

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("activate", namespace),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	artifactCacheDir := filepath.Join(ts.Dirs.Cache, constants.ArtifactMetaDir)
	suite.True(fileutils.DirExists(artifactCacheDir), "artifact cache directory does not exist")
	artifactInfoJson := filepath.Join(artifactCacheDir, constants.ArtifactCacheFileName)
	suite.True(fileutils.FileExists(artifactInfoJson), "artifact cache info json file does not exist")

	files, err := fileutils.ListDir(artifactCacheDir, false)
	suite.NoError(err)
	suite.True(len(files) > 1, "artifact cache is empty") // ignore json file

	// Clear all cached data except artifact cache.
	// This removes the runtime so that it needs to be created again.
	files, err = fileutils.ListDir(ts.Dirs.Cache, true)
	suite.NoError(err)
	for _, entry := range files {
		if entry.IsDir() && entry.RelativePath() != constants.ArtifactMetaDir {
			os.RemoveAll(entry.Path())
		}
	}

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("activate", namespace),
		e2e.OptAppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			"VERBOSE=true", // Necessary to assert "Fetched cached artifact"
		),
	)

	cp.Expect("Fetched cached artifact")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}
