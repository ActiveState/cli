package clean

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
)

func (suite *CleanTestSuite) TestCache() {
	runner := newCache(&testOutputer{}, &confirmMock{confirm: true})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().NoError(err)
	time.Sleep(2 * time.Second)

	if fileutils.DirExists(suite.cachePath) {
		suite.Fail("cache directory should not exist after clean cache")
	}
	if !fileutils.DirExists(suite.configPath) {
		suite.Fail("config directory should exist after clean cache")
	}
	if !fileutils.FileExists(suite.installPath) {
		suite.Fail("installed file should exist after clean cache")
	}
}

func (suite *CleanTestSuite) TestCache_PromptNo() {
	runner := newCache(&testOutputer{}, &confirmMock{})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().NoError(err)

	suite.Require().DirExists(suite.configPath)
	suite.Require().DirExists(suite.cachePath)
	suite.Require().FileExists(suite.installPath)
}

func (suite *CleanTestSuite) TestCache_Activated() {
	os.Setenv(constants.ActivatedStateEnvVarName, "true")
	defer func() {
		os.Unsetenv(constants.ActivatedStateEnvVarName)
	}()

	runner := newCache(&testOutputer{}, &confirmMock{})
	runner.path = suite.cachePath
	err := runner.Run(&CacheParams{})
	suite.Require().Error(err)
}
