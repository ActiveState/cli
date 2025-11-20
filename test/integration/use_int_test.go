package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type UseIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UseIntegrationTestSuite) TestUse() {
	suite.OnlyRunForTags(tagsuite.Use)
	suite.SkipUnsupportedArchitectures()

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout.
	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	// Use.
	cp = ts.Spawn("use", "ActiveState-CLI/Python3")
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	// Verify runtime works.
	pythonExe := filepath.Join(ts.Dirs.DefaultBin, "python3"+osutils.ExeExtension)
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Checkout another project.
	cp = ts.Spawn("checkout", "ActiveState-CLI/Python-3.9")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	// Use it.
	cp = ts.Spawn("use", "ActiveState-CLI/Python-3.9")
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	// Verify the new runtime works.
	cp = ts.SpawnCmdWithOpts(pythonExe, e2e.OptArgs("--version"))
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Switch back using just the project name.
	cp = ts.Spawn("use", "Python3")
	cp.Expect("Switched to project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	// Verify the first runtime is set up correctly and usable.
	cp = ts.SpawnCmdWithOpts(pythonExe, e2e.OptArgs("--version"))
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Test failure switching to project name that was not checked out.
	cp = ts.Spawn("use", "NotCheckedOut")
	cp.Expect("Cannot find the NotCheckedOut project.")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *UseIntegrationTestSuite) TestUseCwd() {
	suite.OnlyRunForTags(tagsuite.Use)
	suite.SkipUnsupportedArchitectures()

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	projDir := filepath.Join(ts.Dirs.Work, "MyEmpty")

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Empty", projDir))
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("use"),
		e2e.OptWD(projDir),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	emptyDir := filepath.Join(ts.Dirs.Work, "EmptyDir")
	suite.Require().NoError(fileutils.Mkdir(emptyDir))
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("use"),
		e2e.OptWD(emptyDir),
	)
	cp.Expect("Unable to use project")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *UseIntegrationTestSuite) TestReset() {
	suite.OnlyRunForTags(tagsuite.Use)
	suite.SkipUnsupportedArchitectures()

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.SetupRCFile()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "ActiveState-CLI/Python3")
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	python3Exe := filepath.Join(ts.Dirs.DefaultBin, "python3"+osutils.ExeExtension)
	suite.True(fileutils.TargetExists(python3Exe), python3Exe+" not found")

	cfg, err := config.New()
	suite.NoError(err)
	rcfile, err := subshell.New(cfg).RcFile()
	if runtime.GOOS != "windows" && fileutils.FileExists(rcfile) {
		suite.NoError(err)
		suite.Contains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH does not have your project in it")
	}

	cp = ts.Spawn("use", "reset")
	cp.Expect("Continue?")
	cp.SendLine("n")
	cp.Expect("Reset aborted by user")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("use", "reset", "--non-interactive")
	cp.Expect("Stopped using your project runtime")
	cp.Expect("Note you may need to")
	cp.ExpectExitCode(0)

	suite.False(fileutils.TargetExists(python3Exe), python3Exe+" still exists")

	cp = ts.Spawn("use", "reset")
	cp.Expect("No project to stop using")
	cp.ExpectExitCode(1)

	if runtime.GOOS != "windows" && fileutils.FileExists(rcfile) {
		suite.NotContains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH still has your project in it")
	}
}

func (suite *UseIntegrationTestSuite) TestShow() {
	suite.OnlyRunForTags(tagsuite.Use)
	suite.SkipUnsupportedArchitectures()

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("use", "show")
	cp.Expect("No project is being used")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "ActiveState-CLI/Empty")
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "show")
	cp.Expect("The active project is ActiveState-CLI/Empty")
	projectDir := filepath.Join(ts.Dirs.Work, "Empty")
	if runtime.GOOS != "windows" {
		cp.Expect(projectDir)
	} else {
		// Windows uses the long path here.
		longPath, err := fileutils.GetLongPathName(projectDir)
		suite.Require().NoError(err)
		cp.Expect(longPath)
	}
	cp.Expect(ts.Dirs.Cache)
	cp.Expect("exec")
	cp.ExpectExitCode(0)

	err := os.RemoveAll(projectDir)
	suite.Require().NoError(err)

	cp = ts.Spawn("use", "show")
	cp.Expect("Cannot find your project")
	// Both Windows and MacOS can run into path comparison issues with symlinks and long paths.
	if runtime.GOOS == "linux" {
		cp.Expect(fmt.Sprintf("Could not find project at %s", projectDir))
	}
	cp.ExpectExitCode(1)

	cp = ts.Spawn("use", "reset", "--non-interactive")
	cp.Expect("Stopped using your project runtime")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "show")
	cp.Expect("No project is being used")
	cp.ExpectExitCode(1)
}

func (suite *UseIntegrationTestSuite) TestSetupNotice() {
	suite.OnlyRunForTags(tagsuite.Use)
	suite.SkipUnsupportedArchitectures()

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect(locale.T("install_runtime"))
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	suite.Require().NoError(os.RemoveAll(filepath.Join(ts.Dirs.Work, "Empty"))) // runtime marker still exists

	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty#265f9914-ad4d-4e0a-a128-9d4e8c5db820")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "Empty")
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)
}

func (suite *UseIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Use, tagsuite.JSON)
	suite.SkipUnsupportedArchitectures()

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty", ".")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "-o", "json")
	cp.Expect(`"namespace":`)
	cp.Expect(`"path":`)
	cp.Expect(`"executables":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("use", "show", "--output", "json")
	cp.Expect(`"namespace":`)
	cp.Expect(`"path":`)
	cp.Expect(`"executables":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestUseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UseIntegrationTestSuite))
}
