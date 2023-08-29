package integration

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		e2e.OptArgs("checkout", "ActiveState-CLI/Python-3.9", "."),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out project")
	suite.Require().True(fileutils.DirExists(ts.Dirs.Work), "state checkout should have created "+ts.Dirs.Work)
	suite.Require().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out properly")

	// Verify runtime was installed correctly and works.
	targetDir := target.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
	pythonExe := filepath.Join(setup.ExecDir(targetDir), "python3"+exeutils.Extension)
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	suite.Run("Cached", func() {
		artifactCacheDir := filepath.Join(ts.Dirs.Cache, constants.ArtifactMetaDir)
		projectCacheDir := target.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
		suite.Require().NotEmpty(fileutils.ListFilesUnsafe(artifactCacheDir), "Artifact cache dir should have files")
		suite.Require().NotEmpty(fileutils.ListFilesUnsafe(projectCacheDir), "Project cache dir should have files")

		suite.Require().NoError(os.RemoveAll(projectCacheDir))                                    // Ensure we can hit the cache by deleting the cache
		suite.Require().NoError(os.Remove(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))) // Ensure we can do another checkout

		cp = ts.SpawnWithOpts(
			e2e.OptArgs("checkout", "ActiveState-CLI/Python-3.9", "."),
			e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
		cp.Expect("Fetched cached artifact") // Comes from log, which is why we're using VERBOSE=true
		cp.Expect("Checked out project")
		cp.ExpectExitCode(0)
	})
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutNonEmptyDir() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	tmpdir := fileutils.TempDirUnsafe()
	_, err := projectfile.Create(&projectfile.CreateParams{Owner: "foo", Project: "bar", Directory: tmpdir})
	suite.Require().NoError(err, "could not write project file")
	_, err2 := fileutils.WriteTempFile("bogus.txt", []byte("test"))
	suite.Require().NoError(err2, "could not write test file")

	// Checkout and verify.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Python3", tmpdir),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=true"),
	)
	cp.Expect("already a project checked out at")
	cp.ExpectExitCode(1)

	// remove file
	suite.Require().NoError(os.Remove(filepath.Join(tmpdir, constants.ConfigFileName)))
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Python3", tmpdir),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=true"),
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
			e2e.OptArgs("checkout", "ActiveState-CLI/Python3", "."),
			e2e.OptWD(dir),
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
	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Python3", "."))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.Expect(ts.Dirs.Work)
	suite.Assert().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out to the current working directory")

	// Test checkout out to a generic path.
	python3Dir := filepath.Join(ts.Dirs.Work, "MyPython3")
	cp = ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Python3#6d9280e7-75eb-401a-9e71-0d99759fbad3", python3Dir))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.Require().True(fileutils.DirExists(python3Dir), "state checkout should have created "+python3Dir)
	asy := filepath.Join(python3Dir, constants.ConfigFileName)
	suite.Require().True(fileutils.FileExists(asy), "ActiveState-CLI/Python3 was not checked out properly")
	suite.Assert().True(bytes.Contains(fileutils.ReadFileUnsafe(asy), []byte("6d9280e7-75eb-401a-9e71-0d99759fbad3")), "did not check out specific commit ID")

	// Test --branch mismatch in non-checked-out project.
	branchPath := filepath.Join(ts.Dirs.Base, "branch")
	cp = ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Python-3.9", branchPath, "--branch", "doesNotExist"))
	cp.Expect("This project has no branch with label matching doesNotExist")
	cp.ExpectExitCode(1)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutCustomRTPath() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	customRTPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, "custom-cache"))
	suite.Require().NoError(err)
	err = fileutils.Mkdir(customRTPath)
	suite.Require().NoError(err)

	// Checkout and verify.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Python3", fmt.Sprintf("--runtime-path=%s", customRTPath)),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out project")

	pythonExe := filepath.Join(setup.ExecDir(customRTPath), "python3"+exeutils.Extension)
	suite.Require().True(fileutils.DirExists(customRTPath))
	suite.Require().True(fileutils.FileExists(pythonExe))

	// Verify runtime was installed correctly and works.
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Verify that state exec works with custom cache.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "python3", "--", "-c", "import sys;print(sys.executable)"),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		e2e.OptWD(filepath.Join(ts.Dirs.Work, "Python3")),
	)
	if runtime.GOOS == "windows" {
		customRTPath, err = fileutils.GetLongPathName(customRTPath)
		suite.Require().NoError(err)
		customRTPath = strings.ToLower(customRTPath)
	}
	cp.Expect(customRTPath)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutAlreadyCheckedOut() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/small-python"))
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/small-python"))
	cp.Expect("already a project checked out at")
	cp.ExpectNotExitCode(0)
}

func (suite *CheckoutIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Checkout, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/small-python", "-o", "json"))
	cp.Expect(`"namespace":`)
	cp.Expect(`"path":`)
	cp.Expect(`"executables":`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // cannot assert here due to "Skipping runtime setup" notice
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutCaseInsensitive() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ACTIVESTATE-CLI/SMALL-PYTHON"))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, "SMALL-PYTHON", constants.ConfigFileName))
	suite.Assert().Contains(string(bytes), "ActiveState-CLI/small-python", "did not match namespace case")
	suite.Assert().NotContains(string(bytes), "ACTIVESTATE-CLI/SMALL-PYTHON", "kept incorrect namespace case")
}

func TestCheckoutIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CheckoutIntegrationTestSuite))
}
