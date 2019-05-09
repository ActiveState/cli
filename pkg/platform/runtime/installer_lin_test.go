// +build linux

package runtime_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"
)

var FailTest = failures.Type("runtime_test.fail")
var FailureToDownload = FailTest.New("unable to download")

type InstallerLinuxTestSuite struct {
	suite.Suite

	dataDir    string
	installDir string
	installer  *runtime.Installer
	rmock      *rmock.Mock
}

func (suite *InstallerLinuxTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "failure obtaining root path")

	suite.dataDir = path.Join(root, "pkg", "platform", "runtime", "testdata")

	suite.rmock = rmock.Init()
	suite.rmock.MockFullRuntime()

	var fail *failures.Failure
	suite.installer, fail = runtime.InitInstaller()
	suite.Require().NoError(fail.ToError())
	suite.Require().NotNil(suite.installer)

	cachePath := config.CachePath()
	if fileutils.DirExists(cachePath) {
		err := os.RemoveAll(config.CachePath())
		suite.Require().NoError(err)
	}

	// Only linux is supported for now, so force it so we can run this test on mac
	// If we want to skip this on mac it should be skipped through build tags, in
	// which case this tweak is meaningless and only a convenience for when testing manually
	model.OS = sysinfo.Linux
}

func (suite *InstallerLinuxTestSuite) AfterTest(suiteName, testName string) {
	suite.rmock.Close()
	err := os.RemoveAll(suite.installDir)
	suite.Require().NoError(err, "failure removing working dir")
	for _, installDir := range suite.installer.InstallDirs() {
		err := os.RemoveAll(installDir)
		if err != nil {
			logging.Warningf("Could not remove installDir: %v\n", err)
		}
	}
}

func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveDoesNotExist() {
	fail := suite.installer.InstallFromArchives([]string{"/no/such/archive.tar.gz"})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), fail.Error())
}

func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveNotTarGz() {
	invalidArchive := path.Join(suite.dataDir, "empty.archive")

	file, fail := fileutils.Touch(invalidArchive)
	suite.Require().NoError(fail.ToError())
	suite.Require().NoError(file.Close())

	fail = suite.installer.InstallFromArchives([]string{invalidArchive})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), fail.Error())
}

func (suite *InstallerLinuxTestSuite) TestInstall_BadArchive() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "badarchive.tar.gz")})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Contains(fail.Error(), "EOF")
}

func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveHasNoInstallDir_ForTarGz() {
	archivePath := path.Join(suite.dataDir, "empty.tar.gz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveNoInstallDir, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestInstall_RuntimeHasNoInstallDir_ForTgz() {
	archivePath := path.Join(suite.dataDir, "empty.tgz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveNoInstallDir, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestInstall_RuntimeMissingPythonExecutable() {
	archivePath := path.Join(suite.dataDir, "python-missing-python-binary.tar.gz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailRuntimeNoExecutable, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestInstall_PythonFoundButNotExecutable() {
	archivePath := path.Join(suite.dataDir, "python-noexec-python.tar.gz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailRuntimeNotExecutable, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestInstall_InstallerFailsToGetPrefixes() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "python-fail-prefixes.tar.gz")})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailRuntimeNoPrefixes, fail.Type)
}

func (suite *InstallerLinuxTestSuite) testRelocation(archive string, executable string) {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, archive)})
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(suite.installer.InstallDirs(), "Installs artifacts")

	suite.Require().True(fileutils.DirExists(suite.installer.InstallDirs()[0]), "expected install-dir to exist")

	pathToExecutable := filepath.Join(suite.installer.InstallDirs()[0], "bin", executable)
	suite.Require().True(fileutils.FileExists(pathToExecutable), executable+" exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.installer.InstallDirs()[0], "bin", "a-script")))
	suite.Contains(ascriptContents, pathToExecutable)
}

func (suite *InstallerLinuxTestSuite) TestInstall_Python_RelocationSuccessful() {
	testRelocation("python-good-installer.tar.gz", constants.ActivePython3Executable)
}

func (suite *InstallerLinuxTestSuite) TestInstall_Python_Legacy_RelocationSuccessful() {
	testRelocation("python-good-installer-nometa.tar.gz", constants.ActivePython3Executable)
}

func (suite *InstallerLinuxTestSuite) TestInstall_Perl_RelocationSuccessful() {
	testRelocation("perl-good-installer.tar.gz", constants.ActivePerlExecutable)
}

func (suite *InstallerLinuxTestSuite) TestInstall_Perl_Legacy_RelocationSuccessful() {
	testRelocation("perl-good-installer-nometa.tar.gz", constants.ActivePerlExecutable)
}

func (suite *InstallerLinuxTestSuite) TestInstall_EventsCalled() {
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

func (suite *InstallerLinuxTestSuite) TestInstall_LegacyAndNew() {
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

func Test_InstallerLinuxTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerLinuxTestSuite))
}
