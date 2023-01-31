package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	cp.Expect("Checked out project")
	python3Dir := filepath.Join(ts.Dirs.Work, "Python3")
	suite.Require().True(fileutils.DirExists(python3Dir), "state checkout should have created "+python3Dir)
	suite.Require().True(fileutils.FileExists(filepath.Join(python3Dir, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out properly")

	// Verify runtime was installed correctly and works.
	targetDir := target.ProjectDirToTargetDir(python3Dir, ts.Dirs.Cache)
	pythonExe := filepath.Join(setup.ExecDir(targetDir), "python3"+exeutils.Extension)
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutNonEmptyDir() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	tmpdir := fileutils.TempDirUnsafe()
	_, err := projectfile.Create(&projectfile.CreateParams{Owner: "foo", Project: "bar", Directory: tmpdir})
	suite.Require().NoError(err, "could not write project file")
	_, err2 := fileutils.WriteTempFile(tmpdir, "active", []byte("test"), 0600)
	suite.Require().NoError(err2, "could not write test file")

	// Checkout and verify.
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python3", tmpdir),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=true"),
	)
	cp.Expect("project at the target path does not match")
	cp.ExpectExitCode(1)

	// remove file
	suite.Require().NoError(os.Remove(filepath.Join(tmpdir, constants.ConfigFileName)))
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("checkout", "ActiveState-CLI/Python3", tmpdir),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=true"),
	)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutMultiDir() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	dirs := []string{
		fileutils.TempDirUnsafe(), fileutils.TempDirUnsafe(),
	}

	for x, dir := range dirs {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("checkout", "ActiveState-CLI/Python3", "."),
			e2e.WithWorkDirectory(dir),
		)
		cp.Expect("Skipping runtime setup")
		cp.Expect("Checked out")
		cp.ExpectExitCode(0)
		suite.Require().FileExists(filepath.Join(dir, constants.ConfigFileName), "Dir %d", x)
	}
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutWithFlags() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Test checking out to current working directory.
	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3", "."))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.Expect(ts.Dirs.Work)
	suite.Assert().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out to the current working directory")

	// Test checkout out to a generic path.
	python3Dir := filepath.Join(ts.Dirs.Work, "MyPython3")
	cp = ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python3#6d9280e7-75eb-401a-9e71-0d99759fbad3", python3Dir))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.Require().True(fileutils.DirExists(python3Dir), "state checkout should have created "+python3Dir)
	asy := filepath.Join(python3Dir, constants.ConfigFileName)
	suite.Require().True(fileutils.FileExists(asy), "ActiveState-CLI/Python3 was not checked out properly")
	suite.Assert().True(bytes.Contains(fileutils.ReadFileUnsafe(asy), []byte("6d9280e7-75eb-401a-9e71-0d99759fbad3")), "did not check out specific commit ID")

	// Test --branch mismatch in non-checked-out project.
	branchPath := filepath.Join(ts.Dirs.Base, "branch")
	cp = ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/Python-3.9", branchPath, "--branch", "doesNotExist"))
	cp.ExpectLongString("This project has no branch with label matching doesNotExist")
	cp.ExpectExitCode(1)

}

func TestCheckoutIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CheckoutIntegrationTestSuite))
}
