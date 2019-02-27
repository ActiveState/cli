package installer_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installer"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/stretchr/testify/suite"
)

type APYInstallerTestSuite struct {
	suite.Suite

	dataDir        string
	baseInstallDir string
}

func (suite *APYInstallerTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "internal", "installer", "testdata")

	suite.baseInstallDir, err = ioutil.TempDir("", "apy-install-test")
	suite.Require().NoError(err, "failure creating working temp dir")
}

func (suite *APYInstallerTestSuite) AfterTest(suiteName, testName string) {
	err := os.RemoveAll(suite.baseInstallDir)
	suite.Require().NoError(err, "failure removing working dir")
}

func (suite *APYInstallerTestSuite) TestNew_WorkingDirDoesNotExist() {
	apyInstaller, failure := installer.NewActivePythonInstaller("/no/such/dir", "/no/such/archive.tar.gz")
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailWorkingDirInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_workingdir_invalid", "/no/such/dir"), failure.Error())
}

func (suite *APYInstallerTestSuite) TestNew_WorkingDirNotADirectory() {
	workingDirFile := path.Join(suite.baseInstallDir, "a.file")

	file, failure := fileutils.Touch(workingDirFile)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := installer.NewActivePythonInstaller(workingDirFile, "/no/such/archive.tar.gz")
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailWorkingDirInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_workingdir_invalid", workingDirFile), failure.Error())
}

func (suite *APYInstallerTestSuite) TestNew_ArchiveDoesNotExist() {
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.baseInstallDir, "/no/such/archive.tar.gz")
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), failure.Error())
}

func (suite *APYInstallerTestSuite) TestNew_ArchiveNotTarGz() {
	invalidArchive := path.Join(suite.baseInstallDir, "archive.file")

	file, failure := fileutils.Touch(invalidArchive)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := installer.NewActivePythonInstaller(suite.baseInstallDir, invalidArchive)
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailArchiveInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), failure.Error())
}

func (suite *APYInstallerTestSuite) TestNew_DistributionDirAlreadyExists() {
	distDir := path.Join(suite.baseInstallDir, constants.ActivePythonDistsDir, "empty")
	failure := fileutils.MkdirUnlessExists(distDir)
	suite.Require().Nil(failure, "trying to precreate dist-dir")

	archivePath := path.Join(suite.dataDir, "empty.tar.gz")
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.baseInstallDir, archivePath)
	suite.Require().Nil(apyInstaller)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInstallation, failure.Type)
	suite.Equal(locale.Tr("installer_err_dist_already_exists", "empty"), failure.Error())
}

func (suite *APYInstallerTestSuite) newInstaller(archivePath string) *installer.ActivePythonInstaller {
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.baseInstallDir, archivePath)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)
	return apyInstaller
}

func (suite *APYInstallerTestSuite) TestNew_Success() {
	archivePath := path.Join(suite.dataDir, "apy-good-installer.tar.gz")
	apyInstaller := suite.newInstaller(archivePath)
	suite.Implements((*installer.Installer)(nil), apyInstaller)
	suite.Equal("apy-good-installer", apyInstaller.DistributionName())
	suite.Equal(path.Join(suite.baseInstallDir, constants.ActivePythonDistsDir, "apy-good-installer"), apyInstaller.DistributionDir())
	suite.Equal(archivePath, apyInstaller.ArchivePath())
}

func (suite *APYInstallerTestSuite) TestInstall_BadArchive() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "badarchive.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailArchiveInvalid, failure.Type)
	suite.Contains(failure.Error(), "EOF")
}

func (suite *APYInstallerTestSuite) TestInstall_ArchiveHasNoInstallDir_ForTarGz() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "empty.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_dist_missing_install_dir", apyInstaller.ArchivePath(), path.Join("empty", "INSTALLDIR")), failure.Error())
	suite.False(fileutils.DirExists(path.Join(path.Dir(apyInstaller.DistributionDir()), constants.ActivePythonInstallDir)), "interim install-dir still exists")
	suite.False(fileutils.DirExists(apyInstaller.DistributionDir()), "dist-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_DistHasNoInstallDir_ForTgz() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "empty.tgz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_dist_missing_install_dir", apyInstaller.ArchivePath(), path.Join("empty", "INSTALLDIR")), failure.Error())
	suite.False(fileutils.DirExists(path.Join(path.Dir(apyInstaller.DistributionDir()), constants.ActivePythonInstallDir)), "interim install-dir still exists")
	suite.False(fileutils.DirExists(apyInstaller.DistributionDir()), "dist-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_DistMissingPythonExecutable() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-missing-python-binary.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_dist_no_executable", apyInstaller.ArchivePath(), constants.ActivePythonExecutable), failure.Error())
	suite.False(fileutils.DirExists(apyInstaller.DistributionDir()), "dist-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_PythonFoundButNotExecutable() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-noexec-python.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailDistInvalid, failure.Type)

	suite.Equal(locale.Tr("installer_err_dist_executable_not_exec", apyInstaller.ArchivePath(), constants.ActivePythonExecutable), failure.Error())
	suite.False(fileutils.DirExists(apyInstaller.DistributionDir()), "dist-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_InstallerFailsToGetPrefixes() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-fail-prefixes.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().NotNil(failure)
	suite.Require().Equal(installer.FailDistInvalid, failure.Type)
	suite.Equal(locale.Tr("installer_err_fail_obtain_prefixes", "apy-fail-prefixes"), failure.Error())

	suite.False(fileutils.DirExists(apyInstaller.DistributionDir()), "dist-dir still exists")
}

func (suite *APYInstallerTestSuite) TestInstall_RelocationSuccessful() {
	apyInstaller := suite.newInstaller(path.Join(suite.dataDir, "apy-good-installer.tar.gz"))
	failure := apyInstaller.Install()
	suite.Require().Nil(failure)

	suite.Require().True(fileutils.DirExists(apyInstaller.DistributionDir()), "expected dist dir to exist")

	// assert relocation prefixes were extracted
	installLog := path.Join(apyInstaller.DistributionDir(), "install.log")
	suite.Require().True(fileutils.FileExists(installLog), "expected test-only install log to be created")
	logContents := string(fileutils.ReadFileUnsafe(installLog))
	suite.Contains(logContents, "import activestate; print(*activestate.prefixes, sep='\\n')")
	suite.Contains(logContents, "success")

	// assert files in installation go relocated
	pathToPython := path.Join(apyInstaller.DistributionDir(), "bin", constants.ActivePythonExecutable)

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(apyInstaller.DistributionDir(), "bin", "a-script")))
	suite.Contains(ascriptContents, pathToPython)

	fooPyLib := string(fileutils.ReadFileUnsafe(path.Join(apyInstaller.DistributionDir(), "lib", "foo.py")))
	suite.Contains(fooPyLib, pathToPython)
}

func Test_APYInstallerTestSuite(t *testing.T) {
	suite.Run(t, new(APYInstallerTestSuite))
}
