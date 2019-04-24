// +build linux

package runtime_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

var FailTest = failures.Type("runtime_test.fail")
var FailureToDownload = FailTest.New("unable to download")

type APYInstallerTestSuite struct {
	suite.Suite

	dataDir    string
	installDir string
}

func (suite *APYInstallerTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "pkg", "platform", "runtime", "testdata")

	suite.installDir, err = ioutil.TempDir("", "apy-install-test")
	suite.Require().NoError(err, "failure creating working temp dir")
}

func (suite *APYInstallerTestSuite) AfterTest(suiteName, testName string) {
	err := os.RemoveAll(suite.installDir)
	suite.Require().NoError(err, "failure removing working dir")
}

func (suite *APYInstallerTestSuite) newInstaller() runtime.Installer {
	apyInstaller, failure := runtime.InitActivePythonInstaller(suite.installDir)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)
	return apyInstaller
}

func (suite *APYInstallerTestSuite) TestInit_InstallDirNotADirectory() {
	workingDirFile := path.Join(suite.installDir, "a.file")

	file, failure := fileutils.Touch(workingDirFile)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := runtime.InitActivePythonInstaller(workingDirFile)
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailInstallDirInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_installdir_isfile", workingDirFile), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInit_InstallDirCreatedIfDoesNotExist() {
	suite.Require().NoError(os.RemoveAll(suite.installDir))
	suite.Require().False(fileutils.DirExists(suite.installDir), "install-dir should have been removed")

	suite.newInstaller()
	suite.True(fileutils.DirExists(suite.installDir), "install-dir should have been created")
}

func (suite *APYInstallerTestSuite) TestInit_Success() {
	apyInstaller := suite.newInstaller()
	suite.Implements((*runtime.Installer)(nil), apyInstaller)
	suite.Equal(suite.installDir, apyInstaller.InstallDir())
}

func (suite *APYInstallerTestSuite) TestInstall_DownloadFails() {
	mockDownloader := rmock.NewMockDownloader()
	mockDownloader.On("Download").Return("", FailureToDownload)
	apyInstaller, failure := runtime.NewActivePythonInstaller(suite.installDir, mockDownloader)
	suite.Require().NotNil(apyInstaller)
	suite.Require().Nil(failure)

	suite.Equal(FailureToDownload, apyInstaller.Install())

	mockDownloader.AssertExpectations(suite.T())
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveDoesNotExist() {
	apyInstaller, failure := runtime.InitActivePythonInstaller(suite.installDir)
	suite.Require().NotNil(apyInstaller)
	suite.Require().Nil(failure)

	failure = apyInstaller.InstallFromArchive("/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveNotTarGz() {
	apyInstaller, failure := runtime.InitActivePythonInstaller(suite.installDir)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)

	invalidArchive := path.Join(suite.dataDir, "empty.archive")

	file, failure := fileutils.Touch(invalidArchive)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	failure = apyInstaller.InstallFromArchive(invalidArchive)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_BadArchive() {
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(path.Join(suite.dataDir, "badarchive.tar.gz"))
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailArchiveInvalid, failure.Type)
	suite.Contains(failure.Error(), "EOF")
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveHasNoInstallDir_ForTarGz() {
	archivePath := path.Join(suite.dataDir, "empty.tar.gz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(archivePath)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_runtime_missing_install_dir", archivePath, path.Join("empty", "INSTALLDIR")), failure.Error())
	suite.False(fileutils.DirExists(path.Join(path.Dir(apyInstaller.InstallDir()), constants.ActivePythonInstallDir)), "interim install-dir still exists")
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "install-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RuntimeHasNoInstallDir_ForTgz() {
	archivePath := path.Join(suite.dataDir, "empty.tgz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(path.Join(suite.dataDir, "empty.tgz"))
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_runtime_missing_install_dir", archivePath, path.Join("empty", "INSTALLDIR")), failure.Error())
	suite.False(fileutils.DirExists(path.Join(path.Dir(apyInstaller.InstallDir()), constants.ActivePythonInstallDir)), "interim install-dir still exists")
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "install-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RuntimeMissingPythonExecutable() {
	archivePath := path.Join(suite.dataDir, "apy-missing-python-binary.tar.gz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(archivePath)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_runtime_no_executable", archivePath, constants.ActivePython2Executable, constants.ActivePython3Executable), failure.Error())
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "install-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_PythonFoundButNotExecutable() {
	archivePath := path.Join(suite.dataDir, "apy-noexec-python.tar.gz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(archivePath)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_runtime_executable_not_exec", archivePath, constants.ActivePython3Executable), failure.Error())
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "install-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_InstallerFailsToGetPrefixes() {
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(path.Join(suite.dataDir, "apy-fail-prefixes.tar.gz"))
	suite.Require().NotNil(failure)
	suite.Require().Equal(runtime.FailRuntimeInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_fail_obtain_prefixes", "apy-fail-prefixes"), failure.Error())

	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "install-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RelocationSuccessful() {
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.InstallFromArchive(path.Join(suite.dataDir, "apy-good-installer.tar.gz"))
	suite.Require().Nil(failure)

	suite.Require().True(fileutils.DirExists(apyInstaller.InstallDir()), "expected install-dir to exist")

	// make sure apy-good-installer and sub-dirs (e.g. INSTALLDIR) gets removed
	suite.False(fileutils.DirExists(path.Join(apyInstaller.InstallDir(), "apy-good-installer")),
		"expected INSTALLDIR not to exist in install-dir")

	// assert files in installation get relocated
	pathToPython3 := path.Join(apyInstaller.InstallDir(), "bin", constants.ActivePython3Executable)
	suite.Require().True(fileutils.FileExists(pathToPython3), "python3 exists")
	suite.Require().True(
		fileutils.FileExists(path.Join(apyInstaller.InstallDir(), "bin", "python")),
		"python hard-link exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(apyInstaller.InstallDir(), "bin", "a-script")))
	suite.Contains(ascriptContents, pathToPython3)

	fooPyLib := string(fileutils.ReadFileUnsafe(path.Join(apyInstaller.InstallDir(), "lib", "foo.py")))
	suite.Contains(fooPyLib, pathToPython3)
}

func (suite *APYInstallerTestSuite) TestInstall_EventsCalled() {
	runtimeMock := rmock.Init()
	runtimeMock.MockFullRuntime()
	defer runtimeMock.Close()

	pjfile := &projectfile.Project{
		Name:  "string",
		Owner: "string",
	}
	pjfile.Persist()

	apyInstaller := suite.newInstaller()

	onDownloadCalled := false
	onInstallCalled := false

	apyInstaller.OnDownload(func() { onDownloadCalled = true })
	apyInstaller.OnInstall(func() { onInstallCalled = true })

	apyInstaller.Install()

	suite.True(onDownloadCalled, "OnDownload is triggered")
	suite.True(onInstallCalled, "OnInstall is triggered")
}

func Test_APYInstallerTestSuite(t *testing.T) {
	suite.Run(t, new(APYInstallerTestSuite))
}
