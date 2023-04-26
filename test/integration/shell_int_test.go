package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ShellIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShellIntegrationTestSuite) TestShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/small-python"),
	)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	args := []string{"small-python", "ActiveState-CLI/small-python"}
	for _, arg := range args {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("shell", arg),
		)
		cp.Expect("Activated")
		cp.WaitForInput()

		cp.SendLine("python3 --version")
		cp.Expect("Python 3")
		cp.SendLine("exit")
		cp.Expect("Deactivated")
		cp.ExpectExitCode(0)
	}

	// Both Windows and MacOS can run into path comparison issues with symlinks and long paths.
	projectName := "small-python"
	if runtime.GOOS == "linux" {
		projectDir := filepath.Join(ts.Dirs.Work, projectName)
		// projectDir, err := fileutils.SymlinkTarget(projectDir)
		// suite.Require().NoError(err)
		err := os.RemoveAll(projectDir)
		suite.Require().NoError(err)

		cp = ts.Spawn("shell", projectName)
		cp.ExpectLongString(fmt.Sprintf("Could not load project %s from path: %s", projectName, projectDir))
	}

	// Check for project not checked out.
	args = []string{"Python-3.9", "ActiveState-CLI/Python-3.9"}
	for _, arg := range args {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("shell", arg),
		)
		cp.Expect("Cannot find the Python-3.9 project")
		cp.ExpectExitCode(1)
	}
}

func (suite *ShellIntegrationTestSuite) TestDefaultShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout.
	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/small-python"))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	// Use.
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/small-python"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestCwdShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell"),
		e2e.WithWorkDirectory(filepath.Join(ts.Dirs.Work, "small-python")),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestCd() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	subdir := filepath.Join(ts.Dirs.Work, "foo", "bar", "baz")
	err := fileutils.Mkdir(subdir)
	suite.Require().NoError(err)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell", "ActiveState-CLI/small-python"),
		e2e.WithWorkDirectory(subdir),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	if runtime.GOOS != "windows" {
		cp.SendLine("pwd")
	} else {
		cp.SendLine("echo %cd%")
	}
	cp.ExpectLongString(subdir)
	cp.SendLine("exit")

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell", "ActiveState-CLI/small-python", "--cd"),
		e2e.WithWorkDirectory(subdir),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	if runtime.GOOS != "windows" {
		cp.SendLine("ls")
	} else {
		cp.SendLine("dir")
	}
	cp.Expect("activestate.yaml")
	cp.SendLine("exit")

	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestDefaultNoLongerExists() {
	suite.OnlyRunForTags(tagsuite.Shell)

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

	err := os.RemoveAll(filepath.Join(ts.Dirs.Work, "Python3"))
	suite.Require().NoError(err)

	cp = ts.SpawnWithOpts(e2e.WithArgs("shell"))
	cp.Expect("Cannot find your project")
	cp.ExpectExitCode(1)
}

func (suite *ShellIntegrationTestSuite) TestUseShellUpdates() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	// Create a zsh RC file
	var zshRcFile string
	var err error
	if runtime.GOOS != "windows" {
		zsh := &zsh.SubShell{}
		zshRcFile, err = zsh.RcFile()
		suite.NoError(err)
		err = fileutils.TouchFileUnlessExists(zshRcFile)
		suite.NoError(err)
	}

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("SHELL=bash"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	// Ensure both bash and zsh RC files are updated
	cfg, err := config.New()
	suite.NoError(err)
	rcfile, err := subshell.New(cfg).RcFile()
	if runtime.GOOS != "windows" {
		suite.NoError(err)
		suite.Contains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH does not have your project in it")
		suite.Contains(string(fileutils.ReadFileUnsafe(zshRcFile)), ts.Dirs.DefaultBin, "PATH does not have your project in it")
	}
}

func (suite *ShellIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Shell, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Perl-5.32", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell", "-o", "json"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect(`{"ACTIVESTATE_ACTIVATED":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestShellIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShellIntegrationTestSuite))
}
