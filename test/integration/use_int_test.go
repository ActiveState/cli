package integration

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
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

	projectsDir := filepath.Join(ts.Dirs.Base, "projects")
	suite.Assert().False(fileutils.DirExists(projectsDir), "projects dir should not exist yet")

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			"ACTIVESTATE_CLI_PROJECTSDIR="+projectsDir),
	)
	cp.Expect("Switched to Python3")
	suite.Require().True(fileutils.DirExists(projectsDir), "projects dir should exist now")
	python3Dir := filepath.Join(projectsDir, "Python3")
	suite.Require().True(fileutils.DirExists(python3Dir), "state use should have created "+python3Dir)
	suite.Require().True(fileutils.FileExists(filepath.Join(python3Dir, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out properly")

	pythonExe := filepath.Join(ts.Dirs.DefaultBin, "python3")
	if runtime.GOOS == "windows" {
		pythonExe = pythonExe + ".bat"
	}
	cp = ts.SpawnCmdWithOpts(
		pythonExe,
		e2e.WithArgs("--version"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Python 3.6.6")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python-3.9"),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			"ACTIVESTATE_CLI_PROJECTSDIR="+projectsDir),
	)
	cp.Expect("Switched to Python-3.9")
	python39Dir := filepath.Join(projectsDir, "Python-3.9")
	suite.Require().True(fileutils.DirExists(python39Dir), "state use should have created "+python39Dir)
	suite.Require().True(fileutils.FileExists(filepath.Join(python39Dir, constants.ConfigFileName)), "project was not checked out properly")

	python3ASY := filepath.Join(projectsDir, "Python3", constants.ConfigFileName)
	python3ASYModTime, err := fileutils.ModTime(python3ASY)
	suite.Assert().NoError(err)
	python39ASY := filepath.Join(projectsDir, "Python-3.9", constants.ConfigFileName)
	python39ASYModTime, err := fileutils.ModTime(python39ASY)
	suite.Assert().NoError(err)
	suite.Assert().True(python39ASYModTime.After(python3ASYModTime), "Python-3.9 accidentally overwrote Python3. Oops.")

	cp = ts.SpawnCmdWithOpts(
		pythonExe,
		e2e.WithArgs("--version"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Python 3.9.10")
	cp.ExpectExitCode(0)

	// Switch back using just the project name and it should not re-checkout.
	timeNow := time.Now()
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("use", "Python3"),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			"ACTIVESTATE_CLI_PROJECTSDIR="+projectsDir),
	)
	cp.Expect("Switched to Python3")
	python3ModTime, err := fileutils.ModTime(pythonExe)
	suite.Require().NoError(err)
	suite.Assert().True(python3ModTime.Unix() <= timeNow.Unix()+1, "ActiveState-CLI/Python3 was checked out again instead of reused")

	cp = ts.SpawnCmdWithOpts(
		pythonExe,
		e2e.WithArgs("--version"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Python 3.6.6")
	cp.ExpectExitCode(0)
}

func (suite *UseIntegrationTestSuite) TestReset() {
	suite.OnlyRunForTags(tagsuite.Use)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectExitCode(0)

	python3Exe := filepath.Join(ts.Dirs.DefaultBin, "python3"+osutils.ExeExt)
	if runtime.GOOS == "windows" {
		python3Exe = python3Exe + ".bat"
	}
	suite.True(fileutils.TargetExists(python3Exe), python3Exe+" not found")

	cfg, err := config.New()
	suite.NoError(err)
	rcfile, err := subshell.New(cfg).RcFile()
	if runtime.GOOS != "windows" {
		suite.NoError(err)
		suite.Contains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH does not have default project in it")
	}

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset"))
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Reset default project runtime")
	cp.Expect("Note you may need to")
	cp.ExpectExitCode(0)

	suite.False(fileutils.TargetExists(python3Exe), python3Exe+" still exists")

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset", "-f"))
	cp.Expect("No global default project to reset")
	cp.ExpectExitCode(0)

	if runtime.GOOS != "windows" {
		suite.NotContains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH still has default project in it")
	}
}

func TestUseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UseIntegrationTestSuite))
}
