package runtime_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	rt "runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	pmock "github.com/ActiveState/cli/internal/progress/mock"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type InstallerTestSuite struct {
	suite.Suite

	dataDir     string
	cacheDir    string
	downloadDir string
	installer   *runtime.Installer
	rmock       *rmock.Mock
	prg         *pmock.TestProgress
}

func (suite *InstallerTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "pkg", "platform", "runtime", "testdata")

	suite.rmock = rmock.Init()
	suite.rmock.MockFullRuntime()

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	suite.cacheDir, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)

	suite.downloadDir, err = ioutil.TempDir("", "cli-installer-test-download")
	suite.Require().NoError(err)

	suite.prg = pmock.NewTestProgress()
	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstallerByParams(runtime.NewInstallerParams(suite.cacheDir, "00010001-0001-0001-0001-000100010001", "string", "string"))
	suite.Require().NoError(fail.ToError())
	suite.Require().NotNil(suite.installer)
}

func (suite *InstallerTestSuite) AfterTest(suiteName, testName string) {
	suite.rmock.Close()
	if err := os.RemoveAll(suite.cacheDir); err != nil {
		logging.Warningf("Could not remove runtimeDir: %v\n", err)
	}
	if err := os.RemoveAll(suite.downloadDir); err != nil {
		logging.Warningf("Could not remove downloadDir: %v\n", err)
	}
	suite.prg.Close()
}

func (suite *InstallerTestSuite) testRelocation(archiveName string, executable string) {
	archive := archiveName + camelInstallerExtension()

	artifact, archives := headchefArtifact(path.Join(suite.dataDir, archive))

	envGetter, fail := runtime.NewCamelRuntime([]*runtime.HeadChefArtifact{artifact}, suite.cacheDir)
	suite.Require().NoError(fail.ToError(), "camel runtime assembler initialized")
	suite.Require().NotEmpty(suite.cacheDir, "Installs artifacts")

	fail = suite.installer.InstallFromArchives(archives, envGetter, suite.prg.Progress)
	suite.Require().NoError(fail.ToError())

	suite.prg.AssertProperClose(suite.T())

	pathToExecutable := filepath.Join(suite.cacheDir, "bin", executable)
	suite.Require().FileExists(pathToExecutable, executable+" exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.cacheDir, "bin", "a-script")))
	suite.Contains(ascriptContents, pathToExecutable)
}

func (suite *InstallerTestSuite) TestInstall_Python_RelocationSuccessful() {
	suite.testRelocation("python-good-installer", constants.ActivePython3Executable)
}

func (suite *InstallerTestSuite) TestInstall_Python_Legacy_RelocationSuccessful() {
	if rt.GOOS == "darwin" {
		suite.T().Skip("Our macOS Python builds do not use relocation, so this will fail if it has to auto detect relocation paths")
	}
	suite.testRelocation("python-good-installer-nometa", constants.ActivePython3Executable)
}

func (suite *InstallerTestSuite) TestInstall_Perl_RelocationSuccessful() {
	suite.testRelocation("perl-good-installer", constants.ActivePerlExecutable)
}

func (suite *InstallerTestSuite) TestInstall_Perl_Legacy_RelocationSuccessful() {
	if rt.GOOS == "darwin" {
		suite.T().Skip("PERL NOT YET SUPPORTED ON MAC")
		return
	}
	suite.testRelocation("perl-good-installer-nometa", constants.ActivePerlExecutable)
}

func Test_InstallerTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerTestSuite))
}
