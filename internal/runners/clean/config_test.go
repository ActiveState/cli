package clean

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
)

func (suite *CleanTestSuite) TestConfig() {
	runner := newConfig(&outputhelper.TestOutputer{}, &confirmMock{confirm: true})
	runner.path = suite.configPath
	err := runner.Run(&ConfigParams{})
	suite.Require().NoError(err)
	time.Sleep(2 * time.Second)

	if fileutils.DirExists(suite.configPath) {
		suite.Fail("config directory should not exist after clean config")
	}
	if !fileutils.DirExists(suite.cachePath) {
		suite.Fail("cache directory should exist after clean config")
	}
	if !fileutils.FileExists(suite.installPath) {
		suite.Fail("installed file should exist after clean config")
	}
}

func (suite *CleanTestSuite) TestConfig_PromptNo() {
	runner := newConfig(&outputhelper.TestOutputer{}, &confirmMock{})
	runner.path = suite.configPath
	err := runner.Run(&ConfigParams{})
	suite.Require().NoError(err)

	suite.Require().DirExists(suite.configPath)
	suite.Require().DirExists(suite.cachePath)
	suite.Require().FileExists(suite.installPath)
}

func (suite *CleanTestSuite) TestConfig_Activated() {
	os.Setenv(constants.ActivatedStateEnvVarName, "true")
	defer func() {
		os.Unsetenv(constants.ActivatedStateEnvVarName)
	}()

	runner := newConfig(&outputhelper.TestOutputer{}, &confirmMock{})
	runner.path = suite.configPath
	err := runner.Run(&ConfigParams{})
	suite.Require().Error(err)
}
