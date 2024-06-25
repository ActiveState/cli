package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("shell", "ActiveState-CLI/Python3")
	cp.Expect("Activated")
	cp.ExpectInput()

	cp.SendLine("python3 -c \"import sys; print(sys.copyright)\"")
	cp.Expect("ActiveState Software Inc.")

	cp.SendLine("exit")
	cp.Expect("Deactivated")
	cp.ExpectExitCode(0)
}

func (suite *ExecutorIntegrationTestSuite) TestExecutorExitCode() {
	suite.OnlyRunForTags(tagsuite.Executor)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("shell", "ActiveState-CLI/Python3")
	cp.Expect("Activated")
	cp.ExpectInput()

	cp.SendLine("python3 -c \"exit(42)\"")

	cp.SendLine("exit")
	cp.ExpectExitCode(42)
}

func sizeByMegs(megabytes float64) int64 {
	return int64(megabytes * float64(1000000))
}

func (suite *ExecutorIntegrationTestSuite) TestExecutorSizeOnDisk() {
	suite.OnlyRunForTags(tagsuite.Executor)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	execFilePath := filepath.Join(ts.Dirs.Bin, constants.StateExecutorCmd+osutils.ExeExtension)
	fi, err := os.Stat(execFilePath)
	suite.Require().NoError(err, "should be able to obtain executor file info")

	maxSize := sizeByMegs(4)
	suite.Require().LessOrEqual(fi.Size(), maxSize, "executor (%d bytes) should be less than or equal to %d bytes", fi.Size(), maxSize)
}
