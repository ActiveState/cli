// +build linux

package runtime_test

// Tests in this file apply to all platforms, but mocking them again for each individual platform is a waste of time.
// It's fairly reliable to say that if a test here succeeds on linux it'll succeed on other platforms, and if it fails
// it'll fail on other platforms.
// I'm sure there'll be exceptions, but for the moment it just isn't worth the timesink to mock these for each platform.

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	pMock "github.com/ActiveState/cli/internal/progress/mock"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type InstallerLinuxTestSuite struct {
	suite.Suite

	cacheDir   string
	dataDir    string
	installDir string
	installer  *runtime.Installer
	rmock      *rmock.Mock
}

func (suite *InstallerLinuxTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "pkg", "platform", "runtime", "testdata")

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	suite.cacheDir, err = ioutil.TempDir("", "cli-installer-test-cache")
	suite.Require().NoError(err)

	suite.installDir, err = ioutil.TempDir("", "cli-installer-test-install")
	suite.Require().NoError(err)

	msgHandler := runbits.NewRuntimeMessageHandler(&outputhelper.TestOutputer{})
	r, err := runtime.NewRuntime("", "00010001-0001-0001-0001-000100010001", "string", "string", msgHandler)
	suite.Require().NoError(err)
	r.SetInstallPath(suite.cacheDir)
	suite.installer = runtime.NewInstaller(r)
	suite.Require().NotNil(suite.installer)
}

func (suite *InstallerLinuxTestSuite) AfterTest(suiteName, testName string) {
	if err := os.RemoveAll(suite.cacheDir); err != nil {
		logging.Warningf("Could not remove runtimeDir: %v\n", err)
	}
	if err := os.RemoveAll(suite.installDir); err != nil {
		logging.Warningf("Could not remove installDir: %v\n", err)
	}
}

func (suite *InstallerLinuxTestSuite) setMocks(a *rmock.Assembler, unpackingOk bool) {
	a.On("PreInstall").Return(nil)
	a.On("PreUnpackArtifact", mock.Anything).Return(nil)
	a.On("Unarchiver").Return(unarchiver.NewTarGz())
	if unpackingOk {
		a.On("PostUnpackArtifact").Return(nil)
	}
}
func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveDoesNotExist() {
	prg := pMock.NewTestProgress()
	defer prg.Close()
	mockAssembler := new(rmock.Assembler)
	suite.setMocks(mockAssembler, false)
	_, archives := headchefArtifact("/no/such/archive.tar.gz")
	fail := suite.installer.InstallFromArchives(archives, mockAssembler, prg.Progress)

	prg.AssertCloseAfterCancellation(suite.T())

	mockAssembler.AssertExpectations(suite.T())

	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), fail.Error())
}

func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveNotTarGz() {
	prg := pMock.NewTestProgress()
	defer prg.Close()

	invalidArchive := path.Join(suite.dataDir, "empty.archive")

	fail := fileutils.Touch(invalidArchive)
	suite.Require().NoError(fail.ToError())

	mockAssembler := new(rmock.Assembler)
	suite.setMocks(mockAssembler, false)

	_, archives := headchefArtifact(invalidArchive)

	fail = suite.installer.InstallFromArchives(archives, mockAssembler, prg.Progress)

	mockAssembler.AssertExpectations(suite.T())

	prg.AssertCloseAfterCancellation(suite.T())
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), fail.Error())
}

func (suite *InstallerLinuxTestSuite) TestInstall_BadArchive() {
	prg := pMock.NewTestProgress()
	defer prg.Close()

	badArchive := path.Join(suite.dataDir, "badarchive.tar.gz")
	mockAssembler := new(rmock.Assembler)
	suite.setMocks(mockAssembler, false)

	_, archives := headchefArtifact(badArchive)
	fail := suite.installer.InstallFromArchives(archives, mockAssembler, prg.Progress)

	prg.AssertCloseAfterCancellation(suite.T())

	mockAssembler.AssertExpectations(suite.T())
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Contains(fail.Error(), "EOF")
}

func Test_InstallerLinuxTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerLinuxTestSuite))
}
