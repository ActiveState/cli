package integration

import (
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

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to Python3")

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
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to Python-3.9")

	cp = ts.SpawnCmdWithOpts(
		pythonExe,
		e2e.WithArgs("--version"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Python 3.9.10")
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
	cp.Expect("Reset default project runtime")
	cp.Expect("Note you may need to")
	cp.ExpectExitCode(0)

	suite.False(fileutils.TargetExists(python3Exe), python3Exe+" still exists")

	cp = ts.SpawnWithOpts(e2e.WithArgs("use", "reset"))
	cp.Expect("No global default project to reset")
	cp.ExpectExitCode(0)

	if runtime.GOOS != "windows" {
		suite.NotContains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH still has default project in it")
	}
}

func TestUseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UseIntegrationTestSuite))
}
