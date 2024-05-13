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
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type CheckoutIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutPython() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout and verify.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Python-3.9", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checking out project: ActiveState-CLI/Python-3.9")
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("All dependencies have been installed and verified", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	suite.Require().True(fileutils.DirExists(ts.Dirs.Work), "state checkout should have created "+ts.Dirs.Work)
	suite.Require().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out properly")

	// Verify runtime was installed correctly and works.
	targetDir := target.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
	pythonExe := filepath.Join(setup.ExecDir(targetDir), "python3"+osutils.ExeExtension)
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
			e2e.OptAppendEnv(
				constants.DisableRuntime+"=false",
				"VERBOSE=true", // Necessary to assert "Fetched cached artifact"
			),
		)
		cp.Expect("Fetched cached artifact", e2e.RuntimeSourcingTimeoutOpt) // Comes from log, which is why we're using VERBOSE=true
		cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
		cp.ExpectExitCode(0)
	})
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutPerl() {
	suite.OnlyRunForTags(tagsuite.Checkout, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout and verify.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Perl-Alternative", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checking out project: ")
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("All dependencies have been installed and verified", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Checked out project")

	// Verify runtime was installed correctly and works.
	targetDir := target.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
	perlExe := filepath.Join(setup.ExecDir(targetDir), "perl"+osutils.ExeExtension)
	cp = ts.SpawnCmd(perlExe, "--version")
	cp.Expect("This is perl")
	cp.ExpectExitCode(0)
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
	)
	cp.Expect("already a project checked out at")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}

	// remove file
	suite.Require().NoError(os.Remove(filepath.Join(tmpdir, constants.ConfigFileName)))
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Python3", tmpdir),
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
	cp.Expect("This project has no branch with label matching 'doesNotExist'")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
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
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	pythonExe := filepath.Join(setup.ExecDir(customRTPath), "python3"+osutils.ExeExtension)
	suite.Require().True(fileutils.DirExists(customRTPath))
	suite.Require().True(fileutils.FileExists(pythonExe))

	// Verify runtime was installed correctly and works.
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Verify that state exec works with custom cache.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "python3", "--", "-c", "import sys;print(sys.executable)"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
		e2e.OptWD(filepath.Join(ts.Dirs.Work, "Python3")),
	)
	if runtime.GOOS == "windows" {
		customRTPath, err = fileutils.GetLongPathName(customRTPath)
		suite.Require().NoError(err)
		customRTPath = strings.ToUpper(customRTPath[:1]) + strings.ToLower(customRTPath[1:]) // capitalize drive letter
	}
	cp.Expect(customRTPath, e2e.RuntimeSourcingTimeoutOpt)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutNotFound() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Bogus-Project-That-Doesnt-Exist"))
	cp.Expect("does not exist under")         // error
	cp.Expect("If this is a private project") // tip
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
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
	ts.IgnoreLogErrors()
}

func (suite *CheckoutIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Checkout, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/small-python", "-o", "json"))
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.SpawnWithOpts(e2e.OptArgs("checkout", "ActiveState-CLI/Bogus-Project-That-Doesnt-Exist", "-o", "json"))
	cp.Expect("does not exist")                        // error
	cp.Expect(`"tips":["If this is a private project`) // tip
	cp.ExpectNotExitCode(0)
	AssertValidJSON(suite.T(), cp)
	ts.IgnoreLogErrors()
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

func (suite *CheckoutIntegrationTestSuite) TestCheckoutBuildtimeClosure() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("Skipping on windows since the build time is different there, and testing it on mac/linux is sufficient")
		return
	}
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/small-python#5a1e49e5-8ceb-4a09-b605-ed334474855b"),
		e2e.OptAppendEnv(constants.InstallBuildDependencies+"=true"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	// Expect the number of build deps to be 27 which is more than the number of runtime deps.
	// Also expect ncurses which should not be in the runtime closure.
	cp.Expect("ncurses", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("27/27", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)
}

func (suite *CheckoutIntegrationTestSuite) TestFail() {
	suite.OnlyRunForTags(tagsuite.Checkout)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/fail"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("failed to build")
	cp.ExpectNotExitCode(0)
	suite.Assert().NoDirExists(filepath.Join(ts.Dirs.Work, "fail"), "state checkout fail did not remove created directory")
	ts.IgnoreLogErrors()

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/fail", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("failed to build")
	cp.ExpectNotExitCode(0)
	suite.Assert().NoFileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName), "state checkout fail did not remove created activestate.yaml")

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/fail", "--force"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("failed to build")
	cp.ExpectNotExitCode(0)
	suite.Assert().DirExists(filepath.Join(ts.Dirs.Work, "fail"), "state checkout fail did not leave created directory there despite --force flag override")
}

func (suite *CheckoutIntegrationTestSuite) TestBranch() {
	suite.OnlyRunForTags(tagsuite.Checkout)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Branches", "--branch", "firstbranch", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	asy := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	suite.Assert().Contains(string(fileutils.ReadFileUnsafe(asy)), "branch=firstbranch", "activestate.yaml does not have correct branch")

	suite.Require().NoError(os.Remove(asy))

	// Infer branch name from commit.
	cp = ts.Spawn("checkout", "ActiveState-CLI/Branches#46c83477-d580-43e2-a0c6-f5d3677517f1", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.Assert().Contains(string(fileutils.ReadFileUnsafe(asy)), "branch=secondbranch", "activestate.yaml does not have correct branch")
}

func TestCheckoutIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CheckoutIntegrationTestSuite))
}
