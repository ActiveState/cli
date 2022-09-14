package integration

import (
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ExecutorIntegrationTestSuite struct {
	tagsuite.Suite
}

func TestExecutorIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorIntegrationTestSuite))
}

func (suite *ExecutorIntegrationTestSuite) TestExecutorForwards() {
	suite.OnlyRunForTags(tagsuite.Executor)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python"),
	)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell", "ActiveState-CLI/Python"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()

	if runtime.GOOS == "linux" {
		cp.SendLine("which python3")
		cp.Expect("python")
		cp.SendLine("echo ${PATH}")
		cp.Expect("bin")
		cp.SendLine("which python3.10")
		cp.Expect("fail")
	}

	cp.SendLine("python3 -c \"import sys; print(sys.copyright)\"")
	cp.Expect("ActiveState Software Inc.")

	cp.SendLine("exit")
	cp.Expect("Deactivated")
	cp.ExpectExitCode(0)
}
