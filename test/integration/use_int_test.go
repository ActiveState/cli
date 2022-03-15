package integration

import (
	"path/filepath"
	"testing"

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

func TestUseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UseIntegrationTestSuite))
}
