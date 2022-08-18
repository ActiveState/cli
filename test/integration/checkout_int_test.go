package integration

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/stretchr/testify/suite"
)

type CheckoutIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CheckoutIntegrationTestSuite) TestCheckout() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout and verify.
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out Python3")
	python3Dir := filepath.Join(ts.Dirs.Work, "Python3")
	suite.Require().True(fileutils.DirExists(python3Dir), "state checkout should have created "+python3Dir)
	suite.Require().True(fileutils.FileExists(filepath.Join(python3Dir, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out properly")

	// Verify runtime was installed correctly and works.
	targetDir := target.ProjectDirToTargetDir(python3Dir, ts.Dirs.Cache)
	pythonExe := filepath.Join(setup.ExecDir(targetDir), "python3")
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
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutWithFlags() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test --path for checking out to current working directory.
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python3", "--path", "."),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out")
	cp.Expect(ts.Dirs.Work)
	suite.Assert().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out to the current working directory")

	// Test --path for checkout out to a generic path.
	python3Dir := filepath.Join(ts.Dirs.Work, "MyPython3")
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python3#6d9280e7-75eb-401a-9e71-0d99759fbad3", "--path", python3Dir),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.Require().True(fileutils.DirExists(python3Dir), "state checkout should have created "+python3Dir)
	asy := filepath.Join(python3Dir, constants.ConfigFileName)
	suite.Require().True(fileutils.FileExists(asy), "ActiveState-CLI/Python3 was not checked out properly")
	suite.Assert().True(bytes.Contains(fileutils.ReadFileUnsafe(asy), []byte("6d9280e7-75eb-401a-9e71-0d99759fbad3")), "did not check out specific commit ID")

	// Test --branch mismatch in non-checked-out project.
	branchPath := filepath.Join(ts.Dirs.Base, "branch")
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python-3.9", "--path", branchPath, "--branch", "doesNotExist"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectLongString("This project has no branch with label matching doesNotExist")
	cp.ExpectExitCode(1)

}

func TestCheckoutIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CheckoutIntegrationTestSuite))
}
