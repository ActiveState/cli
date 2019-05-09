// +build linux

package runtime_test

import (
	"os"
	"path"
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

type InstallerTestSuite struct {
	suite.Suite

	dataDir    string
	installDir string
	installer  *runtime.Installer
	rmock      *rmock.Mock
}

func (suite *InstallerTestSuite) BeforeTest(suiteName, testName string) {
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

func (suite *InstallerTestSuite) AfterTest(suiteName, testName string) {
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

func (suite *InstallerTestSuite) TestInstall_ArchiveDoesNotExist() {
	fail := suite.installer.InstallFromArchives([]string{"/no/such/archive.tar.gz"})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), fail.Error())
}

func (suite *InstallerTestSuite) TestInstall_ArchiveNotTarGz() {
	invalidArchive := path.Join(suite.dataDir, "empty.archive")

	file, fail := fileutils.Touch(invalidArchive)
	suite.Require().NoError(fail.ToError())
	suite.Require().NoError(file.Close())

	fail = suite.installer.InstallFromArchives([]string{invalidArchive})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), fail.Error())
}

func (suite *InstallerTestSuite) TestInstall_BadArchive() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "badarchive.tar.gz")})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Contains(fail.Error(), "EOF")
}

func (suite *InstallerTestSuite) TestInstall_ArchiveHasNoInstallDir_ForTarGz() {
	archivePath := path.Join(suite.dataDir, "empty.tar.gz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveNoInstallDir, fail.Type)
}

func (suite *InstallerTestSuite) TestInstall_RuntimeHasNoInstallDir_ForTgz() {
	archivePath := path.Join(suite.dataDir, "empty.tgz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailArchiveNoInstallDir, fail.Type)
}

func (suite *InstallerTestSuite) TestInstall_RuntimeMissingPythonExecutable() {
	archivePath := path.Join(suite.dataDir, "python-missing-python-binary.tar.gz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailRuntimeNoExecutable, fail.Type)
}

func (suite *InstallerTestSuite) TestInstall_PythonFoundButNotExecutable() {
	archivePath := path.Join(suite.dataDir, "python-noexec-python.tar.gz")
	fail := suite.installer.InstallFromArchives([]string{archivePath})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailRuntimeNotExecutable, fail.Type)
}

func (suite *InstallerTestSuite) TestInstall_InstallerFailsToGetPrefixes() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "python-fail-prefixes.tar.gz")})
	suite.Require().Error(fail.ToError())
	suite.Equal(runtime.FailRuntimeNoPrefixes, fail.Type)
}

func (suite *InstallerTestSuite) TestInstall_Python_RelocationSuccessful() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "python-good-installer.tar.gz")})
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(suite.installer.InstallDirs(), "Installs artifacts")

	suite.Require().True(fileutils.DirExists(suite.installer.InstallDirs()[0]), "expected install-dir to exist")

	// make sure cli-good-installer and sub-dirs (e.g. INSTALLDIR) gets removed
	suite.False(fileutils.DirExists(path.Join(suite.installer.InstallDirs()[0], "python-good-installer")),
		"expected INSTALLDIR not to exist in install-dir")

	// assert files in installation get relocated
	pathToPython3 := path.Join(suite.installer.InstallDirs()[0], "bin", constants.ActivePython3Executable)
	suite.Require().True(fileutils.FileExists(pathToPython3), "python3 exists")
	suite.Require().True(
		fileutils.FileExists(path.Join(suite.installer.InstallDirs()[0], "bin", "python3")),
		"python hard-link exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.installer.InstallDirs()[0], "bin", "a-script")))
	suite.Contains(ascriptContents, pathToPython3)
}

func (suite *InstallerTestSuite) TestInstall_Perl_RelocationSuccessful() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "perl-good-installer.tar.gz")})
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(suite.installer.InstallDirs(), "Installs artifacts")

	suite.Require().True(fileutils.DirExists(suite.installer.InstallDirs()[0]), "expected install-dir to exist")

	// make sure perl-good-installer and sub-dirs (e.g. INSTALLDIR) gets removed
	suite.False(fileutils.DirExists(path.Join(suite.installer.InstallDirs()[0], "perl-good-installer")),
		"expected INSTALLDIR not to exist in install-dir")

	// assert files in installation get relocated
	pathToPerl := path.Join(suite.installer.InstallDirs()[0], "bin", constants.ActivePerlExecutable)
	suite.Require().True(fileutils.FileExists(pathToPerl), "perl exists")
	suite.Require().True(
		fileutils.FileExists(path.Join(suite.installer.InstallDirs()[0], "bin", "perl")),
		"perl hard-link exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.installer.InstallDirs()[0], "bin", "a-script")))
	suite.Contains(ascriptContents, pathToPerl)
}

func (suite *InstallerTestSuite) TestInstall_Perl_Legacy_RelocationSuccessful() {
	fail := suite.installer.InstallFromArchives([]string{path.Join(suite.dataDir, "perl-good-installer-nometa.tar.gz")})
	suite.Require().NoError(fail.ToError())
	suite.Require().NotEmpty(suite.installer.InstallDirs(), "Installs artifacts")

	suite.Require().True(fileutils.DirExists(suite.installer.InstallDirs()[0]), "expected install-dir to exist")

	// make sure perl-good-installer and sub-dirs (e.g. INSTALLDIR) gets removed
	suite.False(fileutils.DirExists(path.Join(suite.installer.InstallDirs()[0], "perl-good-installer-nometa")),
		"expected INSTALLDIR not to exist in install-dir")

	// assert files in installation get relocated
	pathToPerl := path.Join(suite.installer.InstallDirs()[0], "bin", constants.ActivePerlExecutable)
	suite.Require().True(fileutils.FileExists(pathToPerl), "perl exists")
	suite.Require().True(
		fileutils.FileExists(path.Join(suite.installer.InstallDirs()[0], "bin", "perl")),
		"perl hard-link exists")

	ascriptContents := string(fileutils.ReadFileUnsafe(path.Join(suite.installer.InstallDirs()[0], "bin", "a-script")))
	suite.Contains(ascriptContents, pathToPerl)
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
