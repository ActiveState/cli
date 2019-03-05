package runtime_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/stretchr/testify/suite"
)

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

func (suite *APYInstallerTestSuite) newInstaller() *runtime.ActivePythonInstaller {
	apyInstaller, failure := runtime.NewActivePythonInstaller(suite.installDir)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)
	return apyInstaller
}

func (suite *APYInstallerTestSuite) TestNew_InstallDirNotADirectory() {
	workingDirFile := path.Join(suite.installDir, "a.file")

	file, failure := fileutils.Touch(workingDirFile)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := runtime.NewActivePythonInstaller(workingDirFile)
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailInstallDirInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_installdir_isfile", workingDirFile), failure.Error())
}

func (suite *APYInstallerTestSuite) TestNew_InstallDirCreatedIfDoesNotExist() {
	suite.Require().NoError(os.RemoveAll(suite.installDir))
	suite.Require().False(fileutils.DirExists(suite.installDir), "install-dir should have been removed")

	suite.newInstaller()
	suite.True(fileutils.DirExists(suite.installDir), "install-dir should have been created")
}

func (suite *APYInstallerTestSuite) TestNew_RuntimeAlreadyInstalled() {
	f, failure := fileutils.Touch(path.Join(suite.installDir, "regular-file"))
	suite.Require().Nil(failure, "trying to touch a file in the install-dir")
	defer os.Remove(f.Name())

	apyInstaller, failure := runtime.NewActivePythonInstaller(suite.installDir)
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInstallation, failure.Type)
	suite.Equal(locale.Tr("installer_err_runtime_already_exists", suite.installDir), failure.Error())
}

func (suite *APYInstallerTestSuite) TestNew_Success() {
	apyInstaller := suite.newInstaller()
	suite.Implements((*runtime.Installer)(nil), apyInstaller)
	suite.Equal(suite.installDir, apyInstaller.InstallDir())
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveDoesNotExist() {
	apyInstaller, failure := runtime.NewActivePythonInstaller(suite.installDir)
	suite.Require().NotNil(apyInstaller)
	suite.Require().Nil(failure)

	failure = apyInstaller.Install("/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveNotTarGz() {
	apyInstaller, failure := runtime.NewActivePythonInstaller(suite.installDir)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)

	invalidArchive := path.Join(suite.installDir, "archive.file")

	file, failure := fileutils.Touch(invalidArchive)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	failure = apyInstaller.Install(invalidArchive)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_BadArchive() {
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(path.Join(suite.dataDir, "badarchive.tar.gz"))
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailArchiveInvalid, failure.Type)
	suite.Contains(failure.Error(), "EOF")
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveHasNoInstallDir_ForTarGz() {
	archivePath := path.Join(suite.dataDir, "empty.tar.gz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(archivePath)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_runtime_missing_install_dir", archivePath, path.Join("empty", "INSTALLDIR")), failure.Error())
	suite.False(fileutils.DirExists(path.Join(path.Dir(apyInstaller.InstallDir()), constants.ActivePythonInstallDir)), "interim install-dir still exists")
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "runtime-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RuntimeHasNoInstallDir_ForTgz() {
	archivePath := path.Join(suite.dataDir, "empty.tgz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(path.Join(suite.dataDir, "empty.tgz"))
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_runtime_missing_install_dir", archivePath, path.Join("empty", "INSTALLDIR")), failure.Error())
	suite.False(fileutils.DirExists(path.Join(path.Dir(apyInstaller.InstallDir()), constants.ActivePythonInstallDir)), "interim install-dir still exists")
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "runtime-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RuntimeMissingPythonExecutable() {
	archivePath := path.Join(suite.dataDir, "apy-missing-python-binary.tar.gz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(archivePath)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_runtime_no_executable", archivePath, constants.ActivePythonExecutable), failure.Error())
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "runtime-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_PythonFoundButNotExecutable() {
	archivePath := path.Join(suite.dataDir, "apy-noexec-python.tar.gz")
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(archivePath)
	suite.Require().NotNil(failure)
	suite.Equal(runtime.FailRuntimeInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_runtime_executable_not_exec", archivePath, constants.ActivePythonExecutable), failure.Error())
	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "runtime-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_InstallerFailsToGetPrefixes() {
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(path.Join(suite.dataDir, "apy-fail-prefixes.tar.gz"))
	suite.Require().NotNil(failure)
	suite.Require().Equal(runtime.FailRuntimeInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_fail_obtain_prefixes", "apy-fail-prefixes"), failure.Error())

	suite.False(fileutils.DirExists(apyInstaller.InstallDir()), "runtime-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RelocationSuccessful() {
	apyInstaller := suite.newInstaller()
	failure := apyInstaller.Install(path.Join(suite.dataDir, "apy-good-installer.tar.gz"))
	suite.Require().Nil(failure)

	suite.Require().True(fileutils.DirExists(apyInstaller.InstallDir()), "expected runtime dir to exist")

	// make sure INSTALLDIR gets removed
	suite.False(fileutils.DirExists(path.Join(apyInstaller.InstallDir(), constants.ActivePythonInstallDir)),
		"expected INSTALLDIR not to exist in runtime-dir")

	// assert files in installation get relocated
	pathToPython := path.Join(apyInstaller.InstallDir(), "bin", constants.ActivePythonExecutable)

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(apyInstaller.InstallDir(), "bin", "a-script")))
	suite.Contains(ascriptContents, pathToPython)

	fooPyLib := string(fileutils.ReadFileUnsafe(path.Join(apyInstaller.InstallDir(), "lib", "foo.py")))
	suite.Contains(fooPyLib, pathToPython)
}

func Test_APYInstallerTestSuite(t *testing.T) {
	suite.Run(t, new(APYInstallerTestSuite))
}
