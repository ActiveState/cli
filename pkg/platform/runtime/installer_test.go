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

	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstaller(suite.downloadDir, suite.cacheDir, runtime.InitDownload(suite.downloadDir))
	suite.Require().NoError(fail.ToError())
	suite.Require().NotNil(suite.installer)
}

func (suite *InstallerTestSuite) AfterTest(suiteName, testName string) {
	suite.rmock.Close()
	if err := os.RemoveAll(suite.cacheDir); err != nil {
		logging.Warningf("Could not remove cacheDir: %v\n", err)
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
	suite.Require().NotEmpty(envGetter.InstallDirs(), "Installs artifacts")

	fail = suite.installer.InstallFromArchives(archives, envGetter, suite.prg.Progress)
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(suite.installer.InstallDirs(), "Installs artifacts")

	suite.Require().True(fileutils.DirExists(suite.installer.InstallDirs()[0]), "expected install-dir to exist")

	pathToExecutable := filepath.Join(suite.installer.InstallDirs()[0], "bin", executable)
	suite.Require().FileExists(pathToExecutable)

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.installer.InstallDirs()[0], "bin", "a-script")))
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

func (suite *InstallerTestSuite) TestInstall_EventsCalled() {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	cacheDir, err := ioutil.TempDir("", "")
	suite.Require().NoError(err)

	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstaller(downloadDir, cacheDir, runtime.InitDownload(downloadDir))
	suite.Require().NoError(fail.ToError())

	onDownloadCalled := false

	suite.installer.OnDownload(func() { onDownloadCalled = true })

	envGetter, freshInstall, fail := suite.installer.Install()
	suite.Require().NoError(fail.ToError())
	suite.Assert().NotNil(envGetter)
	suite.Assert().True(freshInstall)

	suite.True(onDownloadCalled, "OnDownload is triggered")

	onDownloadCalled = false
	envGetter, freshInstall, fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())
	suite.Assert().NotNil(envGetter)
	suite.Assert().False(freshInstall)

	suite.False(onDownloadCalled, "OnDownload is not triggered, because we already downloaded it")
}

func (suite *InstallerTestSuite) TestInstall_LegacyAndNew() {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	var fail *failures.Failure
	suite.installer, fail = runtime.InitInstaller()
	suite.Require().NoError(fail.ToError())

	envGetter, freshInstall, fail := suite.installer.Install()
	suite.Require().NoError(fail.ToError())
	suite.Assert().NotNil(envGetter)
	suite.Assert().True(freshInstall)

	suite.Require().Len(suite.installer.InstallDirs(), 2)

	metaCount := 0
	for _, installDir := range suite.installer.InstallDirs() {
		if _, fail := runtime.InitMetaData(installDir); fail == nil {
			metaCount = metaCount + 1
		}
	}

	suite.Equal(2, metaCount, "Both new and legacy got installed via metafile")
}

func Test_InstallerTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerTestSuite))
}
