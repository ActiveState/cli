package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type UseIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UseIntegrationTestSuite) TestUse() {
	suite.OnlyRunForTags(tagsuite.Use)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout.
	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3"))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	// Use.
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	// Verify runtime works.
	pythonExe := filepath.Join(ts.Dirs.DefaultBin, "python3"+osutils.ExeExt)
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Checkout another project.
	cp = ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python-3.9"))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	// Use it.
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python-3.9"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	// Verify the new runtime works.
	cp = ts.SpawnCmdWithOpts(pythonExe, e2e.WithArgs("--version"))
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Switch back using just the project name.
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	// Verify the first runtime is set up correctly and usable.
	cp = ts.SpawnCmdWithOpts(pythonExe, e2e.WithArgs("--version"))
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Test failure switching to project name that was not checked out.
	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "NotCheckedOut"))
	cp.Expect("Cannot find the NotCheckedOut project.")
	cp.ExpectExitCode(1)
}

func (suite *UseIntegrationTestSuite) TestUseCwd() {
	suite.OnlyRunForTags(tagsuite.Use)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	pythonDir := filepath.Join(ts.Dirs.Work, "MyPython3")

	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3", pythonDir))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use"),
		e2e.WithWorkDirectory(pythonDir),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	emptyDir := filepath.Join(ts.Dirs.Work, "EmptyDir")
	suite.Require().NoError(fileutils.Mkdir(emptyDir))
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use"),
		e2e.WithWorkDirectory(emptyDir),
	)
	cp.Expect("Unable to use project")
	cp.ExpectExitCode(1)
}

func (suite *UseIntegrationTestSuite) TestReset() {
	suite.OnlyRunForTags(tagsuite.Use)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3"))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	python3Exe := filepath.Join(ts.Dirs.DefaultBin, "python3"+osutils.ExeExt)
	suite.True(fileutils.TargetExists(python3Exe), python3Exe+" not found")

	cfg, err := config.New()
	suite.NoError(err)
	rcfile, err := subshell.New(cfg).RcFile()
	if runtime.GOOS != "windows" {
		suite.NoError(err)
		suite.Contains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH does not have your project in it")
	}

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset"))
	cp.Expect("Continue?")
	cp.SendLine("n")
	cp.Expect("Reset aborted by user")
	cp.ExpectExitCode(1)

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset", "--non-interactive"))
	cp.Expect("Stopped using your project runtime")
	cp.Expect("Note you may need to")
	cp.ExpectExitCode(0)

	suite.False(fileutils.TargetExists(python3Exe), python3Exe+" still exists")

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset", "-n"))
	cp.Expect("No project to stop using")
	cp.ExpectExitCode(0)

	if runtime.GOOS != "windows" {
		suite.NotContains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH still has your project in it")
	}
}

func (suite *UseIntegrationTestSuite) TestShow() {
	suite.OnlyRunForTags(tagsuite.Use)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("use", "show"))
	cp.Expect("No project is being used")
	cp.ExpectExitCode(1)

	cp = ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3"))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "show"),
	)
	cp.ExpectLongString("The active project is ActiveState-CLI/Python3")
	projectDir := filepath.Join(ts.Dirs.Work, "Python3")
	if runtime.GOOS != "windows" {
		cp.ExpectLongString(projectDir)
	} else {
		// Windows uses the long path here.
		longPath, err := fileutils.GetLongPathName(projectDir)
		suite.Require().NoError(err)
		cp.ExpectLongString(longPath)
	}
	cp.ExpectLongString(ts.Dirs.Cache)
	cp.Expect("exec")
	cp.ExpectExitCode(0)

	err := os.RemoveAll(projectDir)
	suite.Require().NoError(err)

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "show"))
	cp.ExpectLongString("Cannot find your project")
	// Both Windows and MacOS can run into path comparison issues with symlinks and long paths.
	if runtime.GOOS == "linux" {
		cp.ExpectLongString(fmt.Sprintf("Could not find project at %s", projectDir))
	}
	cp.ExpectExitCode(1)

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset", "--non-interactive"))
	cp.Expect("Reset")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "show"))
	cp.Expect("No project is being used")
	cp.ExpectExitCode(1)
}

func TestUseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UseIntegrationTestSuite))
}
