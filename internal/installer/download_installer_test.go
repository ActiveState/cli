package installer_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/failures"

	"github.com/ActiveState/cli/internal/installer"
	rmock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/stretchr/testify/suite"
)

var FailTest = failures.Type("installer_test.fail")
var FailureToDownload = FailTest.New("unable to download")
var FailureToInstall = FailTest.New("unable to install")

type DownloadInstallerTestSuite struct {
	suite.Suite

	mockInstaller  *rmock.Installer
	mockDownloader *rmock.Downloader
}

func (suite *DownloadInstallerTestSuite) BeforeTest(suiteName, testName string) {
	suite.mockInstaller = rmock.NewMockInstaller()
	suite.mockDownloader = rmock.NewMockDownloader()
}

func (suite *DownloadInstallerTestSuite) TestDownloadFails() {
	suite.mockDownloader.On("Download").Return("", FailureToDownload)

	installer := installer.NewRuntimeInstaller(suite.mockDownloader, suite.mockInstaller)
	suite.Equal(FailureToDownload, installer.Install())

	suite.mockDownloader.AssertExpectations(suite.T())
	suite.mockInstaller.AssertNotCalled(suite.T(), "Install", "")
}

func (suite *DownloadInstallerTestSuite) TestInstallFails() {
	suite.mockDownloader.On("Download").Return("runtime-archive.tar.gz", nil)
	suite.mockInstaller.On("Install", "runtime-archive.tar.gz").Return(FailureToInstall)

	installer := installer.NewRuntimeInstaller(suite.mockDownloader, suite.mockInstaller)
	suite.Equal(FailureToInstall, installer.Install())

	suite.mockDownloader.AssertExpectations(suite.T())
	suite.mockInstaller.AssertExpectations(suite.T())
}

func Test_DownloadInstallerTestSuite(t *testing.T) {
	suite.Run(t, new(DownloadInstallerTestSuite))
}
