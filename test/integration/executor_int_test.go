package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/pkg/executors"
	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
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

func (suite *ExecutorIntegrationTestSuite) TestExecutorBatArguments() {
	suite.OnlyRunForTags(tagsuite.Executor)

	if runtime.GOOS != "windows" {
		suite.T().Skip("This test is only for windows")
	}

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	root := environment.GetRootPathUnsafe()
	executorsPath := filepath.Join(ts.Dirs.Work, "executors")
	srcExes := fileutils.ListFilesUnsafe(filepath.Join(root, "test", "integration", "testdata", "batarguments"))
	reportExe := filepath.Join(executorsPath, "report.exe")

	t := executors.NewTarget(constants.ValidZeroUUID, "ActiveState-CLI", "test", "")
	executors := executors.New(executorsPath)
	executors.SetExecutorSrc(ts.ExecutorExe)
	err := executors.Apply(
		svcctl.NewIPCSockPathFromGlobals().String(),
		t,
		osutils.EnvSliceToMap(ts.Env),
		srcExes,
	)
	suite.Require().NoError(err)
	suite.Require().FileExists(reportExe)

	// Force override ACTIVESTATE_CI to false, because communicating with the svc will fail, and if this is true
	// the executor will interrupt.
	// For this test we don't care about the svc communication.
	env := e2e.OptAppendEnv("ACTIVESTATE_CI=false")

	inputs := []string{"a<b", "b>a", "hello world", "&whoami", "imnot|apipe", "%NotAppData%", "^NotEscaped", "(NotAGroup)"}
	outputs := `"` + strings.Join(inputs, `" "`) + `"`
	cp := ts.SpawnCmdWithOpts(reportExe, e2e.OptArgs(inputs...), env)
	cp.Expect(outputs, termtest.OptExpectTimeout(5*time.Second))
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmdWithOpts(reportExe, e2e.OptArgs("&whoami"), env)
	cp.Expect(`"&whoami"`, termtest.OptExpectTimeout(5*time.Second))
	cp.ExpectExitCode(0)

	// Ensure regular arguments aren't quoted
	cp = ts.SpawnCmdWithOpts(reportExe, e2e.OptArgs("ImNormal", "I'm Special", "ImAlsoNormal"), env)
	cp.Expect(`ImNormal "I'm Special" ImAlsoNormal`, termtest.OptExpectTimeout(5*time.Second))
	cp.ExpectExitCode(0)
}
