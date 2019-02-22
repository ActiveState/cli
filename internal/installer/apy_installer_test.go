package installer_test

import (
	"os"
	"path"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installer"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/stretchr/testify/suite"
)

type APYInstallerTestSuite struct {
	suite.Suite

	dataDir    string
	workingDir string
}

func (suite *APYInstallerTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "internal", "installer", "testdata")

	suite.workingDir = path.Join(suite.dataDir, "generated", "installer")
	failure := fileutils.MkdirUnlessExists(suite.workingDir)
	suite.Require().Nil(failure, "failure creating test installer dir")
}

func (suite *APYInstallerTestSuite) AfterTest(suiteName, testName string) {
	err := os.RemoveAll(suite.workingDir)
	suite.Require().NoError(err, "failure removing working dir")
}

func (suite *APYInstallerTestSuite) TestNew_WorkingDirDoesNotExist() {
	apyInstaller, failure := installer.NewActivePythonInstaller("/no/such/dir", "/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailWorkingDirInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_workingdir_invalid", "/no/such/dir"), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_WorkingDirNotADirectory() {
	workingDirFile := path.Join(suite.workingDir, "a.file")

	file, failure := fileutils.Touch(workingDirFile)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := installer.NewActivePythonInstaller(workingDirFile, "/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailWorkingDirInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_workingdir_invalid", workingDirFile), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_ArchiveDoesNotExist() {
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.workingDir, "/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_ArchiveNotTarGz() {
	invalidArchive := path.Join(suite.workingDir, "archive.file")

	file, failure := fileutils.Touch(invalidArchive)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := installer.NewActivePythonInstaller(suite.workingDir, invalidArchive)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) newInstaller(archivePath string) *installer.ActivePythonInstaller {
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.workingDir, archivePath)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)
	return apyInstaller
}

func (suite *APYInstallerTestSuite) TestNew_Success() {
	archivePath := path.Join(suite.dataDir, "apy-good-installer.tar.gz")
	apyInstaller := suite.newInstaller(archivePath)
	suite.Implements((*installer.Installer)(nil), apyInstaller)
	suite.Equal("apy-good-installer", apyInstaller.DistributionName())
	suite.Equal(path.Join(suite.workingDir, installer.ActivePythonDistsDir, "apy-good-installer"), apyInstaller.DistributionDir())
	suite.Equal(archivePath, apyInstaller.ArchivePath())
}

func (suite *APYInstallerTestSuite) TestInstall_BadArchive() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "badarchive.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailArchiveInvalid, failure.Type)
	suite.Contains(failure.Error(), "create new gzip reader: EOF")
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveHasNoValidRootDir_ForTarGz() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "empty.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_dist_missing_root_dir", apyInstaller.ArchivePath(), "empty"), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_DistHasNoValidRootDir_ForTgz() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "empty.tgz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_dist_missing_root_dir", apyInstaller.ArchivePath(), "empty"), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_DistMissingInstallScript() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-missing-script.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_dist_no_install_script", apyInstaller.ArchivePath(), installer.ActivePythonInstallScript), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_DistInstallScriptNotExecutable() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-nonexec-script.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_dist_install_script_no_exec", apyInstaller.ArchivePath(), installer.ActivePythonInstallScript), failure.Error())
}

func (suite *APYInstallerTestSuite) TestInstall_Successful() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-good-installer.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().Nil(failure)

	suite.Require().True(fileutils.DirExists(path.Join(suite.workingDir, installer.ActivePythonDistsDir)),
		"expected python dir under working-dir to exist")
	suite.Require().True(fileutils.DirExists(apyInstaller.DistributionDir()), "expected dist dir to exist")

	installLog := path.Join(apyInstaller.DistributionDir(), "install.log")
	suite.Require().True(fileutils.FileExists(installLog), "expected install log to be created")
	suite.Contains(string(fileutils.ReadFileUnsafe(installLog)), "successful install")
}

func (suite *APYInstallerTestSuite) TestInstall_InstallerFails() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-fail-installer.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Require().Equal(installer.FailDistInstallation, failure.Type)
	suite.Equal(locale.Tr("installer_err_installscript_failed", "apy-fail-installer", "exit status 1"), failure.Error())

	suite.Require().False(fileutils.DirExists(apyInstaller.DistributionDir()), "dist dir should be removed")
}

func Test_APYInstallerTestSuite(t *testing.T) {
	suite.Run(t, new(APYInstallerTestSuite))
}
