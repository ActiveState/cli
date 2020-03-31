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
	"github.com/ActiveState/cli/internal/progress"
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

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	suite.cacheDir, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)

	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstallerByParams(runtime.InstallerParams{
		CacheDir:    suite.cacheDir,
		CommitID:    "00010001-0001-0001-0001-000100010001",
		Owner:       "string",
		ProjectName: "string",
	})
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
	prg := progress.New(progress.WithOutput(nil))
	defer prg.Close()
	fail := suite.installer.InstallFromArchives(headchefArtifact(path.Join(suite.dataDir, archive)), prg)
	suite.Require().NoError(fail.ToError())
	installDirs, fail := suite.installer.InstallDirs()
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(installDirs, "Installs artifacts")

	suite.Require().True(fileutils.DirExists(installDirs[0]), "expected install-dir to exist")

	pathToExecutable := filepath.Join(installDirs[0], "bin", executable)
	suite.Require().FileExists(pathToExecutable)

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(installDirs[0], "bin", "a-script")))
	suite.Contains(ascriptContents, pathToExecutable)
}

func (suite *InstallerTestSuite) TestInstall_Python_RelocationSuccessful() {
	suite.testRelocation("python-good-installer"+runtime.InstallerExtension, constants.ActivePython3Executable)
}

func (suite *InstallerTestSuite) TestInstall_Python_Legacy_RelocationSuccessful() {
	if rt.GOOS == "darwin" {
		suite.T().Skip("Our macOS Python builds do not use relocation, so this will fail if it has to auto detect relocation paths")
	}
	suite.testRelocation("python-good-installer-nometa"+runtime.InstallerExtension, constants.ActivePython3Executable)
}

func (suite *InstallerTestSuite) TestInstall_Perl_RelocationSuccessful() {
	suite.testRelocation("perl-good-installer"+runtime.InstallerExtension, constants.ActivePerlExecutable)
}

func (suite *InstallerTestSuite) TestInstall_Perl_Legacy_RelocationSuccessful() {
	if rt.GOOS == "darwin" {
		suite.T().Skip("PERL NOT YET SUPPORTED ON MAC")
		return
	}
	suite.testRelocation("perl-good-installer-nometa"+runtime.InstallerExtension, constants.ActivePerlExecutable)
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
	suite.installer, fail = runtime.NewInstallerByParams(runtime.InstallerParams{
		CacheDir:    cacheDir,
		CommitID:    "00010001-0001-0001-0001-000100010001",
		Owner:       "string",
		ProjectName: "string",
	})
	suite.Require().NoError(fail.ToError())

	onDownloadCalled := false

	suite.installer.OnDownload(func() { onDownloadCalled = true })

	_, fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())

	suite.True(onDownloadCalled, "OnDownload is triggered")

	onDownloadCalled = false
	_, fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())

	suite.False(onDownloadCalled, "OnDownload is not triggered, because we already downloaded it")
}

func (suite *InstallerTestSuite) TestInstall_LegacyAndNew() {
	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstallerByParams(runtime.InstallerParams{
		CacheDir:    suite.cacheDir,
		CommitID:    "00010001-0001-0001-0001-000100010001",
		Owner:       "string",
		ProjectName: "string",
	})
	suite.Require().NoError(fail.ToError())

	_, fail = suite.installer.Install()
	suite.Require().NoError(fail.ToError())

	installDirs, fail := suite.installer.InstallDirs()
	suite.Require().NoError(fail.ToError())
	suite.Require().Len(installDirs, 2)

	metaCount := 0
	for _, installDir := range installDirs {
		if _, fail := runtime.InitMetaData(installDir); fail == nil {
			metaCount = metaCount + 1
		}
	}

	suite.Equal(2, metaCount, "Both new and legacy got installed via metafile")
}

func (suite *InstallerTestSuite) TestInstall_InstallDirsStandalone() {
	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstallerByParams(runtime.InstallerParams{
		CacheDir:    suite.cacheDir,
		CommitID:    "00010001-0001-0001-0001-000100010001",
		Owner:       "string",
		ProjectName: "string",
	})
	suite.Require().NoError(fail.ToError())

	_, fail = suite.installer.InstallDirsStandalone()
	suite.Equal(runtime.FailRequiresDownload.Name, fail.Type.Name)

	installed, fail := suite.installer.Install()
	suite.Require().NoError(fail.ToError())
	suite.Require().True(installed)

	installDirs, fail := suite.installer.InstallDirsStandalone()
	suite.Require().NoError(fail.ToError())
	suite.Require().Len(installDirs, 2)

	metaCount := 0
	for _, installDir := range installDirs {
		if _, fail := runtime.InitMetaData(installDir); fail == nil {
			metaCount = metaCount + 1
		}
	}

	suite.Equal(2, metaCount, "Both new and legacy got installed via metafile")
}

func Test_InstallerTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerTestSuite))
}
