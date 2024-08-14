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
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	rt "github.com/ActiveState/cli/pkg/runtime"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

type CheckoutIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutPython() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout and verify.
	cp := ts.Spawn("checkout", "ActiveState-CLI/Python-3.9", ".")
	cp.Expect("Checking out project: ActiveState-CLI/Python-3.9")
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("All dependencies have been installed and verified", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	suite.Require().True(fileutils.DirExists(ts.Dirs.Work), "state checkout should have created "+ts.Dirs.Work)
	suite.Require().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Python3 was not checked out properly")

	// Verify runtime was installed correctly and works.
	proj, err := project.FromPath(ts.Dirs.Work)
	suite.Require().NoError(err)
	targetDir := filepath.Join(ts.Dirs.Cache, runtime_helpers.DirNameFromProjectDir(proj.Dir()))
	pythonExe := filepath.Join(rt.ExecutorsPath(targetDir), "python3"+osutils.ExeExtension)
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutPerl() {
	suite.OnlyRunForTags(tagsuite.Checkout, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout and verify.
	cp := ts.Spawn("checkout", "ActiveState-CLI/Perl-Alternative", ".")
	cp.Expect("Checking out project: ")
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("All dependencies have been installed and verified", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Checked out project")

	// Verify runtime was installed correctly and works.
	proj, err := project.FromPath(ts.Dirs.Work)
	suite.Require().NoError(err)

	execPath := rt.ExecutorsPath(filepath.Join(ts.Dirs.Cache, runtime_helpers.DirNameFromProjectDir(proj.Dir())))
	perlExe := filepath.Join(execPath, "perl"+osutils.ExeExtension)

	cp = ts.SpawnCmd(perlExe, "--version")
	cp.Expect("This is perl")
	cp.ExpectExit()
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
	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty", tmpdir)
	cp.Expect("already a project checked out at")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}

	// remove file
	suite.Require().NoError(os.Remove(filepath.Join(tmpdir, constants.ConfigFileName)))
	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty", tmpdir)
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
			e2e.OptArgs("checkout", "ActiveState-CLI/Empty", "."),
			e2e.OptWD(dir),
		)
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
	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty", ".")
	cp.Expect("Checked out")
	cp.Expect(ts.Dirs.Work)
	suite.Assert().True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName)), "ActiveState-CLI/Empty was not checked out to the current working directory")

	// Test checkout out to a generic path.
	projectDir := filepath.Join(ts.Dirs.Work, "MyProject")
	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty#6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8", projectDir)
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.Require().True(fileutils.DirExists(projectDir), "state checkout should have created "+projectDir)
	asy := filepath.Join(projectDir, constants.ConfigFileName)
	suite.Require().True(fileutils.FileExists(asy), "ActiveState-CLI/Empty was not checked out properly")
	suite.Assert().True(bytes.Contains(fileutils.ReadFileUnsafe(asy), []byte("6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")), "did not check out specific commit ID")

	// Test --branch mismatch in non-checked-out project.
	branchPath := filepath.Join(ts.Dirs.Base, "branch")
	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty", branchPath, "--branch", "doesNotExist")
	cp.Expect("This project has no branch with label matching 'doesNotExist'")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutCustomRTPath() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	customRTPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, "custom-cache"))
	suite.Require().NoError(err)
	err = fileutils.Mkdir(customRTPath)
	suite.Require().NoError(err)

	// Checkout and verify.
	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3", fmt.Sprintf("--runtime-path=%s", customRTPath))
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	pythonExe := filepath.Join(rt.ExecutorsPath(customRTPath), "python3"+osutils.ExeExtension)
	suite.Require().True(fileutils.DirExists(customRTPath))
	suite.Require().True(fileutils.FileExists(pythonExe))

	// Verify runtime was installed correctly and works.
	cp = ts.SpawnCmd(pythonExe, "--version")
	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	// Verify that state exec works with custom cache.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "python3", "--", "-c", "import sys;print(sys.executable)"),
		e2e.OptWD(filepath.Join(ts.Dirs.Work, "Python3")),
	)
	cp.Expect(customRTPath, e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExit()
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutNotFound() {
	suite.OnlyRunForTags(tagsuite.Checkout)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Bogus-Project-That-Doesnt-Exist")
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

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("already a project checked out at")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func (suite *CheckoutIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Checkout, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty", "-o", "json")
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("checkout", "ActiveState-CLI/Bogus-Project-That-Doesnt-Exist", "-o", "json")
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

	cp := ts.Spawn("checkout", "ACTIVESTATE-CLI/EMPTY")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, "EMPTY", constants.ConfigFileName))
	suite.Assert().Contains(string(bytes), "ActiveState-CLI/Empty", "did not match namespace case")
	suite.Assert().NotContains(string(bytes), "ACTIVESTATE-CLI/EMPTY", "kept incorrect namespace case")
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
		e2e.OptAppendEnv(constants.InstallBuildDependenciesEnvVarName+"=true"),
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

	cp := ts.Spawn("checkout", "ActiveState-CLI/fail")
	cp.Expect("failed to build")
	cp.ExpectNotExitCode(0)
	suite.Assert().NoDirExists(filepath.Join(ts.Dirs.Work, "fail"), "state checkout fail did not remove created directory")
	ts.IgnoreLogErrors()

	cp = ts.Spawn("checkout", "ActiveState-CLI/fail", ".")
	cp.Expect("failed to build")
	cp.ExpectNotExitCode(0)
	suite.Assert().NoFileExists(filepath.Join(ts.Dirs.Work, constants.ConfigFileName), "state checkout fail did not remove created activestate.yaml")

	cp = ts.Spawn("checkout", "ActiveState-CLI/fail", "--force")
	cp.Expect("failed to build")
	cp.ExpectNotExitCode(0)
	suite.Assert().DirExists(filepath.Join(ts.Dirs.Work, "fail"), "state checkout fail did not leave created directory there despite --force flag override")
}

func (suite *CheckoutIntegrationTestSuite) TestBranch() {
	suite.OnlyRunForTags(tagsuite.Checkout)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty", "--branch", "mingw", ".")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	asy := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	suite.Assert().Contains(string(fileutils.ReadFileUnsafe(asy)), "branch=mingw", "activestate.yaml does not have correct branch")

	suite.Require().NoError(os.Remove(asy))

	// Infer branch name from commit.
	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty#830c81b1-95e7-4de0-b48e-4f4579cba794", ".")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	suite.Assert().Contains(string(fileutils.ReadFileUnsafe(asy)), "branch=mingw", "activestate.yaml does not have correct branch")
}

func (suite *CheckoutIntegrationTestSuite) TestNoLanguage() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("The Window's platform is not available for ActiveState-CLI/langless")
	}

	suite.OnlyRunForTags(tagsuite.Checkout, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/langless", ".")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
}

func (suite *CheckoutIntegrationTestSuite) TestChangeSummary() {
	suite.OnlyRunForTags(tagsuite.Checkout)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/small-python")
	cp.Expect("Resolving Dependencies")
	cp.Expect("Done")
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("└─ python@3.10.10")
	suite.Assert().NotContains(cp.Snapshot(), "├─", "more than one dependency was printed")
	cp.ExpectExitCode(0)
}

func (suite *CheckoutIntegrationTestSuite) TestCheckoutFromArchive() {
	suite.OnlyRunForTags(tagsuite.Checkout)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	root := environment.GetRootPathUnsafe()
	archive := filepath.Join(root, "test", "integration", "testdata", "checkout-from-archive", runtime.GOOS+".tar.gz")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", archive),
		e2e.OptAppendEnv("HTTPS_PROXY=none://"), // simulate lack of network connection
	)
	cp.Expect("Checking out project: ActiveState-CLI/AlmostEmpty")
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("└─ zlib@1.3.1")
	cp.Expect("Sourcing Runtime")
	cp.Expect("Unpacking")
	cp.Expect("Installing")
	cp.Expect("All dependencies have been installed and verified")
	cp.Expect("Checked out project ActiveState-CLI/AlmostEmpty")
	cp.ExpectExitCode(0)

	// Verify the zlib runtime files exist.
	proj, err := project.FromPath(filepath.Join(ts.Dirs.Work, "AlmostEmpty"))
	suite.Require().NoError(err)
	cachePath := filepath.Join(ts.Dirs.Cache, runtime_helpers.DirNameFromProjectDir(proj.Dir()))
	zlibH := filepath.Join(cachePath, "usr", "include", "zlib.h")
	suite.Assert().FileExists(zlibH, "zlib.h does not exist")
}

func TestCheckoutIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CheckoutIntegrationTestSuite))
}
