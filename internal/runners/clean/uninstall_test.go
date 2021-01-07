package clean

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
)

type confirmMock struct {
	confirm bool
}

func (c *confirmMock) Confirm(title, message string, defaultChoice bool) (bool, error) {
	return c.confirm, nil
}

type CleanTestSuite struct {
	suite.Suite
	confirm     *confirmMock
	configPath  string
	cachePath   string
	installPath string
}

func (suite *CleanTestSuite) SetupTest() {
	installFile, err := ioutil.TempFile("", "")
	if err != nil {
		suite.Error(err)
	}
	suite.Require().FileExists(installFile.Name())
	suite.installPath = installFile.Name()

	err = installFile.Close()
	suite.Require().NoError(err)

	suite.configPath, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)
	suite.Require().DirExists(suite.configPath)

	suite.cachePath, err = ioutil.TempDir("", "")
	suite.Require().NoError(err)
	suite.Require().DirExists(suite.cachePath)
}

func (suite *CleanTestSuite) TestUninstall() {
	runner, err := newUninstall(&outputhelper.TestOutputer{}, &confirmMock{confirm: true}, newConfigMock(suite.T(), suite.cachePath, suite.configPath))
	suite.Require().NoError(err)
	runner.installPath = suite.installPath
	err = runner.Run(&UninstallParams{})
	suite.Require().NoError(err)
	time.Sleep(2 * time.Second)

	if fileutils.DirExists(suite.configPath) {
		suite.Fail("config directory should not exist after uninstall")
	}
	if fileutils.DirExists(suite.cachePath) {
		suite.Fail("cache directory should not exist after uninstall")
	}
	if fileutils.FileExists(suite.installPath) {
		suite.Fail("installed file should not exist after uninstall")
	}
}

func (suite *CleanTestSuite) TestUninstall_PromptNo() {
	runner, err := newUninstall(&outputhelper.TestOutputer{}, &confirmMock{}, newConfigMock(suite.T(), suite.cachePath, suite.configPath))
	suite.Require().NoError(err)
	err = runner.Run(&UninstallParams{})
	suite.Require().NoError(err)

	suite.Require().DirExists(suite.configPath)
	suite.Require().DirExists(suite.cachePath)
	suite.Require().FileExists(suite.installPath)
}

func (suite *CleanTestSuite) TestUninstall_Activated() {
	os.Setenv(constants.ActivatedStateEnvVarName, "true")
	defer func() {
		os.Unsetenv(constants.ActivatedStateEnvVarName)
	}()

	runner, err := newUninstall(&outputhelper.TestOutputer{}, &confirmMock{}, &configMock{suite.T(), suite.cachePath, suite.configPath})
	suite.Require().NoError(err)
	err = runner.Run(&UninstallParams{})
	suite.Require().Error(err)
}

func (suite *CleanTestSuite) AfterTest(suiteName, testName string) {
	os.RemoveAll(suite.configPath)
	os.RemoveAll(suite.cachePath)
	os.Remove(suite.installPath)
}

func TestCleanTestSuite(t *testing.T) {
	suite.Run(t, new(CleanTestSuite))
}
