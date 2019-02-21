package installer_test

import (
	"os"
	"path"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"

	"github.com/ActiveState/cli/internal/environment"

	"github.com/ActiveState/cli/internal/installer"
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
	suite.Equal(installer.FailInvalidWorkingDir, failure.Type)
	suite.Equal(locale.Tr("installer_err_invalid_workingdir", "/no/such/dir"), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_WorkingDirNotADirectory() {
	workingDirFile := path.Join(suite.workingDir, "a.file")

	file, failure := fileutils.Touch(workingDirFile)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := installer.NewActivePythonInstaller(workingDirFile, "/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailInvalidWorkingDir, failure.Type)
	suite.Equal(locale.Tr("installer_err_invalid_workingdir", workingDirFile), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_ArchiveDoesNotExist() {
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.workingDir, "/no/such/archive.tar.gz")
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailInvalidArchive, failure.Type)
	suite.Equal(locale.Tr("installer_err_notfound_archive", "/no/such/archive.tar.gz"), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_ArchiveNotTarGz() {
	invalidArchive := path.Join(suite.workingDir, "archive.file")

	file, failure := fileutils.Touch(invalidArchive)
	suite.Require().Nil(failure, "failure touching test file")
	suite.Require().NoError(file.Close(), "failure closing test file")

	apyInstaller, failure := installer.NewActivePythonInstaller(suite.workingDir, invalidArchive)
	suite.Require().NotNil(failure)
	suite.Equal(installer.FailInvalidArchive, failure.Type)
	suite.Equal(locale.Tr("installer_err_badext_archive", invalidArchive), failure.Error())
	suite.Nil(apyInstaller)
}

func (suite *APYInstallerTestSuite) TestNew_Success() {
	archivePath := path.Join(suite.dataDir, "empty.tar.gz")
	apyInstaller, failure := installer.NewActivePythonInstaller(suite.workingDir, archivePath)
	suite.Require().Nil(failure)
	suite.Require().NotNil(apyInstaller)
	suite.Implements((*installer.Installer)(nil), apyInstaller)

	suite.Equal(suite.workingDir, apyInstaller.WorkingDir())
	suite.Equal(archivePath, apyInstaller.ArchivePath())
}

// create tempdir and unpack archive
// create dir ${wdir}/python/${basename(archive)}
// execute install.sh from tempdir and pass `-I ${wdir}/python/${basename(archive)}`
// chdir ${wdir}/python && ln -s ${wdir}/python/${basename(archive)} latest

func Test_APYInstallerTestSuite(t *testing.T) {
	suite.Run(t, new(APYInstallerTestSuite))
}
