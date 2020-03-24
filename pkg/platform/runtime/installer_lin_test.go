// +build linux

package runtime_test

// Tests in this file apply to all platforms, but mocking them again for each individual platform is a waste of time.
// It's fairly reliable to say that if a test here succeeds on linux it'll succeed on other platforms, and if it fails
// it'll fail on other platforms.
// I'm sure there'll be exceptions, but for the moment it just isn't worth the timesink to mock these for each platform.

import (
	"bytes"
	"fmt"
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
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	rmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type InstallerLinuxTestSuite struct {
	suite.Suite

	dataDir     string
	cacheDir    string
	downloadDir string
	installer   *runtime.Installer
	rmock       *rmock.Mock

	prg *progress.Progress
}

func (suite *InstallerLinuxTestSuite) BeforeTest(suiteName, testName string) {
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

	suite.cacheDir, err = ioutil.TempDir("", "cli-installer-test-cache")
	suite.Require().NoError(err)

	suite.downloadDir, err = ioutil.TempDir("", "cli-installer-test-download")
	suite.Require().NoError(err)

	var fail *failures.Failure
	suite.installer, fail = runtime.NewInstaller(suite.downloadDir, suite.cacheDir, runtime.InitDownload(suite.downloadDir))
	suite.Require().NoError(fail.ToError())
	suite.Require().NotNil(suite.installer)
	suite.prg = progress.New(progress.WithOutput(nil))
}

func (suite *InstallerLinuxTestSuite) AfterTest(suiteName, testName string) {
	suite.rmock.Close()
	if err := os.RemoveAll(suite.cacheDir); err != nil {
		logging.Warningf("Could not remove cacheDir: %v\n", err)
	}
	if err := os.RemoveAll(suite.downloadDir); err != nil {
		logging.Warningf("Could not remove downloadDir: %v\n", err)
	}
	suite.prg.Close()
}

func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveDoesNotExist() {
	fail := suite.installer.InstallFromArchives(headchefArtifact("/no/such/archive.tar.gz"), suite.prg)
	suite.Require().Error(fail.ToError())
	suite.prg.Cancel()
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_notfound", "/no/such/archive.tar.gz"), fail.Error())
}

func (suite *InstallerLinuxTestSuite) TestInstall_ArchiveNotTarGz() {
	invalidArchive := path.Join(suite.dataDir, "empty.archive")

	file, fail := fileutils.Touch(invalidArchive)
	suite.Require().NoError(fail.ToError())
	suite.Require().NoError(file.Close())

	fail = suite.installer.InstallFromArchives(headchefArtifact(invalidArchive), suite.prg)
	suite.Require().Error(fail.ToError())
	suite.prg.Cancel()
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Equal(locale.Tr("installer_err_archive_badext", invalidArchive), fail.Error())
}

func (suite *InstallerLinuxTestSuite) TestInstall_BadArchive() {
	fail := suite.installer.InstallFromArchives(headchefArtifact(path.Join(suite.dataDir, "badarchive.tar.gz")), suite.prg)
	suite.Require().Error(fail.ToError())
	suite.prg.Cancel()
	suite.Equal(runtime.FailArchiveInvalid, fail.Type)
	suite.Contains(fail.Error(), "EOF")
}

func (suite *InstallerLinuxTestSuite) TestInstall_RuntimeMissingPythonExecutable() {
	archivePath := path.Join(suite.dataDir, "python-missing-python-binary.tar.gz")
	fail := suite.installer.InstallFromArchives(headchefArtifact(archivePath), suite.prg)
	suite.Require().Error(fail.ToError())
	suite.prg.Cancel()
	suite.Equal(runtime.FailMetaDataNotDetected, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestInstall_PythonFoundButNotExecutable() {
	archivePath := path.Join(suite.dataDir, "python-noexec-python.tar.gz")
	fail := suite.installer.InstallFromArchives(headchefArtifact(archivePath), suite.prg)
	suite.Require().Error(fail.ToError())
	suite.prg.Cancel()
	suite.Equal(runtime.FailRuntimeNotExecutable, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestInstall_InstallerFailsToGetPrefixes() {
	fail := suite.installer.InstallFromArchives(headchefArtifact(path.Join(suite.dataDir, "python-fail-prefixes.tar.gz")), suite.prg)
	suite.Require().Error(fail.ToError())
	suite.prg.Cancel()
	suite.Equal(runtime.FailRuntimeNoPrefixes, fail.Type)
}

func (suite *InstallerLinuxTestSuite) TestRelocate() {
	relocationPrefix := "######################################## RELOCATE ME ########################################"

	fileutils.CopyFile(filepath.Join(suite.dataDir, "relocate/bin/python3"), filepath.Join(suite.cacheDir, "relocate/bin/python3"))

	binary := "relocate/binary"
	fileutils.CopyFile(filepath.Join(suite.dataDir, binary), filepath.Join(suite.cacheDir, binary))

	text := "relocate/text.go"
	fileutils.CopyFile(filepath.Join(suite.dataDir, text), filepath.Join(suite.cacheDir, text))

	// Mock metaData
	metaData := &runtime.MetaData{
		Path:          filepath.Join(suite.cacheDir, "relocate"),
		RelocationDir: relocationPrefix,
		BinaryLocations: []runtime.MetaDataBinary{
			runtime.MetaDataBinary{
				Path:     "bin",
				Relative: true,
			},
		},
		Env: map[string]string{},
	}

	metaData.Prepare()
	suite.Equal("lib", metaData.RelocationTargetBinaries)

	installDir := filepath.Join(suite.cacheDir, "relocate")
	upb := suite.prg.AddUnpackBar(10000, 50)
	upb.ReScale(3)

	fail := suite.installer.Relocate(metaData, upb)
	suite.Require().NoError(fail.ToError())

	// test text
	suite.Contains(string(fileutils.ReadFileUnsafe(filepath.Join(suite.cacheDir, text))), fmt.Sprintf("-- %s --", installDir))

	// test binary
	libDir := filepath.Join(suite.cacheDir, "relocate/lib")
	binaryData := fileutils.ReadFileUnsafe(filepath.Join(suite.cacheDir, binary))
	suite.True(len(bytes.Split(binaryData, []byte(libDir))) > 1, "Correctly injects "+libDir)
}

func Test_InstallerLinuxTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerLinuxTestSuite))
}
