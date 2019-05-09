// +build !darwin

package runtime_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
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
}

func (suite *InstallerTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "pkg", "platform", "runtime", "testdata")

	suite.rmock = rmock.Init()
	suite.rmock.MockFullRuntime()

	pjfile := projectfile.Project{
		Name:  "string",
		Owner: "string",
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
}

func (suite *InstallerTestSuite) testRelocation(archive string, executable string) {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, archive)})
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(suite.installer.InstallDirs(), "Installs artifacts")

	suite.Require().True(fileutils.DirExists(suite.installer.InstallDirs()[0]), "expected install-dir to exist")

	pathToExecutable := filepath.Join(suite.installer.InstallDirs()[0], "bin", executable)
	suite.Require().True(fileutils.FileExists(pathToExecutable), executable+" exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.installer.InstallDirs()[0], "bin", "a-script")))
	suite.Contains(ascriptContents, pathToExecutable)
}

func (suite *InstallerTestSuite) TestInstall_Python_RelocationSuccessful() {
	suite.testRelocation("python-good-installer"+runtime.InstallerExtension, constants.ActivePython3Executable)
}

func (suite *InstallerTestSuite) TestInstall_Python_Legacy_RelocationSuccessful() {
	suite.testRelocation("python-good-installer-nometa"+runtime.InstallerExtension, constants.ActivePython3Executable)
}

func (suite *InstallerTestSuite) TestInstall_Perl_RelocationSuccessful() {
	suite.testRelocation("perl-good-installer"+runtime.InstallerExtension, constants.ActivePerlExecutable)
}

func (suite *InstallerTestSuite) TestInstall_Perl_Legacy_RelocationSuccessful() {
	suite.testRelocation("perl-good-installer-nometa"+runtime.InstallerExtension, constants.ActivePerlExecutable)
}

func (suite *InstallerTestSuite) TestInstall_EventsCalled() {
	pjfile := &projectfile.Project{
		Name:  "string",
		Owner: "string",
	}
	pjfile.Persist()

	var fail *failures.Failure
	suite.installer, fail = runtime.InitInstaller()
	suite.Require().NoError(fail.ToError())

	onDownloadCalled := false
	onInstallCalled := false

	suite.installer.OnDownload(func() { onDownloadCalled = true })
	suite.installer.OnInstall(func() { onInstallCalled = true })

	fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())

	suite.True(onDownloadCalled, "OnDownload is triggered")
	suite.True(onInstallCalled, "OnInstall is triggered")

	onDownloadCalled = false
	onInstallCalled = false
	fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())

	suite.False(onDownloadCalled, "OnDownload is not triggered, because we already downloaded it")
	suite.False(onInstallCalled, "OnInstall is not triggered, because we already installed it")
}

func (suite *InstallerTestSuite) TestInstall_LegacyAndNew() {
	pjfile := &projectfile.Project{
		Name:  "string",
		Owner: "string",
	}
	pjfile.Persist()

	var fail *failures.Failure
	suite.installer, fail = runtime.InitInstaller()
	suite.Require().NoError(fail.ToError())

	fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())

	suite.Require().Len(suite.installer.InstallDirs(), 2)

	metaCount := 0
	for _, installDir := range suite.installer.InstallDirs() {
		if _, fail := runtime.InitMetaData(installDir); fail == nil {
			metaCount = metaCount + 1
		}
	}

	suite.Equal(1, metaCount, "Installed one artifact via metafile")
}

func Test_InstallerTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerTestSuite))
}
